// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.opencensus.io/stats"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/appmain/contextcause"
	"open-match.dev/open-match/internal/ipb"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/internal/statestore"
	"open-match.dev/open-match/pkg/pb"
)

// The service implementing the Backend API that is called to generate matches
// and make assignments for Tickets.
type backendService struct {
	synchronizer *synchronizerClient
	store        statestore.Service
	cc           *rpc.ClientCache
}

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.backend",
	})
	errBackfillGenerationMismatch = errors.New("backfill generation mismatch")
)

// FetchMatches triggers a MatchFunction with the specified MatchProfiles, while each MatchProfile
// returns a set of match proposals. FetchMatches method streams the results back to the caller.
// FetchMatches immediately returns an error if it encounters any execution failures.
//   - If the synchronizer is enabled, FetchMatch will then call the synchronizer to deduplicate proposals with overlapped tickets.
func (s *backendService) FetchMatches(req *pb.FetchMatchesRequest, stream pb.BackendService_FetchMatchesServer) error {
	if req.Config == nil {
		return status.Error(codes.InvalidArgument, ".config is required")
	}
	if req.Profile == nil {
		return status.Error(codes.InvalidArgument, ".profile is required")
	}

	// Error group for handling the synchronizer calls only.
	eg, ctx := errgroup.WithContext(stream.Context())
	syncStream, err := s.synchronizer.synchronize(ctx)
	if err != nil {
		return err
	}

	// The mmf must be canceled if the synchronizer call fails (which will
	// cancel the context from the error group).  However the synchronizer call
	// is NOT dependant on the mmf call.
	mmfCtx, cancelMmfs := contextcause.WithCancelCause(ctx)
	// Closed when mmfs should start.
	startMmfs := make(chan struct{})
	proposals := make(chan *pb.Match)
	m := &sync.Map{}

	eg.Go(func() error {
		return synchronizeSend(ctx, syncStream, m, proposals)
	})
	eg.Go(func() error {
		return synchronizeRecv(ctx, syncStream, m, stream, startMmfs, cancelMmfs, s.store)
	})

	var mmfErr error
	select {
	case <-mmfCtx.Done():
		mmfErr = fmt.Errorf("mmf was never started")
	case <-startMmfs:
		mmfErr = callMmf(mmfCtx, s.cc, req, proposals)
	}

	syncErr := eg.Wait()

	// TODO: Send mmf error in FetchSummary instead of erroring call.
	if syncErr != nil || mmfErr != nil {
		return fmt.Errorf(
			"error(s) in FetchMatches call. syncErr=[%v], mmfErr=[%v]",
			syncErr,
			mmfErr,
		)
	}

	return nil
}

func synchronizeSend(ctx context.Context, syncStream synchronizerStream, m *sync.Map, proposals <-chan *pb.Match) error {
sendProposals:
	for {
		select {
		case <-ctx.Done():
			break sendProposals
		case p, ok := <-proposals:
			if !ok {
				break sendProposals
			}
			_, loaded := m.LoadOrStore(p.GetMatchId(), p)
			if loaded {
				return fmt.Errorf("MatchMakingFunction returned same match_id twice: \"%s\"", p.GetMatchId())
			}
			err := syncStream.Send(&ipb.SynchronizeRequest{Proposal: p})
			if err != nil {
				return fmt.Errorf("error sending proposal to synchronizer: %w", err)
			}
		}
	}

	err := syncStream.CloseSend()
	if err != nil {
		return fmt.Errorf("error closing send stream of proposals to synchronizer: %w", err)
	}
	return nil
}

func synchronizeRecv(ctx context.Context, syncStream synchronizerStream, m *sync.Map, stream pb.BackendService_FetchMatchesServer, startMmfs chan<- struct{}, cancelMmfs contextcause.CancelErrFunc, store statestore.Service) error {
	var startMmfsOnce sync.Once

	for {
		resp, err := syncStream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("error receiving match from synchronizer: %w", err)
		}

		if resp.StartMmfs {
			go startMmfsOnce.Do(func() {
				close(startMmfs)
			})
		}

		if resp.CancelMmfs {
			cancelMmfs(errors.New("match function ran longer than proposal window, canceling"))
		}

		if v, ok := m.Load(resp.GetMatchId()); ok {
			match, ok := v.(*pb.Match)
			if !ok {
				return fmt.Errorf("error casting sync map value into *pb.Match: %w", err)
			}

			err = createOrUpdateBackfill(ctx, match, store)
			if err != nil {
				if err == errBackfillGenerationMismatch {
					continue
				}
				return errors.Wrapf(err, "failed to handle match backfill: %s", match.MatchId)
			}

			stats.Record(ctx, totalBytesPerMatch.M(int64(proto.Size(match))))
			stats.Record(ctx, ticketsPerMatch.M(int64(len(match.GetTickets()))))
			err = stream.Send(&pb.FetchMatchesResponse{Match: match})
			if err != nil {
				return fmt.Errorf("error sending match to caller of backend: %w", err)
			}
		}
	}
}

// callMmf triggers execution of MMFs to fetch match proposals.
func callMmf(ctx context.Context, cc *rpc.ClientCache, req *pb.FetchMatchesRequest, proposals chan<- *pb.Match) error {
	defer close(proposals)
	address := fmt.Sprintf("%s:%d", req.GetConfig().GetHost(), req.GetConfig().GetPort())

	switch req.GetConfig().GetType() {
	case pb.FunctionConfig_GRPC:
		return callGrpcMmf(ctx, cc, req.GetProfile(), address, proposals)
	case pb.FunctionConfig_REST:
		return callHTTPMmf(ctx, cc, req.GetProfile(), address, proposals)
	default:
		return status.Error(codes.InvalidArgument, "provided match function type is not supported")
	}
}

func callGrpcMmf(ctx context.Context, cc *rpc.ClientCache, profile *pb.MatchProfile, address string, proposals chan<- *pb.Match) error {
	var conn *grpc.ClientConn
	conn, err := cc.GetGRPC(address)
	if err != nil {
		return status.Error(codes.InvalidArgument, "failed to establish grpc client connection to match function")
	}
	client := pb.NewMatchFunctionClient(conn)

	stream, err := client.Run(ctx, &pb.RunRequest{Profile: profile})
	if err != nil {
		err = errors.Wrap(err, "failed to run match function for profile")
		if ctx.Err() != nil {
			// gRPC likes to suppress the context's error, so stop that.
			return ctx.Err()
		}
		return err
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			err = errors.Wrapf(err, "%v.Run() error, %v", client, err)
			if ctx.Err() != nil {
				// gRPC likes to suppress the context's error, so stop that.
				return ctx.Err()
			}
			return err
		}
		select {
		case proposals <- resp.GetProposal():
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func callHTTPMmf(ctx context.Context, cc *rpc.ClientCache, profile *pb.MatchProfile, address string, proposals chan<- *pb.Match) error {
	client, baseURL, err := cc.GetHTTP(address)
	if err != nil {
		err = errors.Wrapf(err, "failed to establish rest client connection to match function: %s", address)
		return status.Error(codes.InvalidArgument, err.Error())
	}

	var m jsonpb.Marshaler
	strReq, err := m.MarshalToString(&pb.RunRequest{Profile: profile})
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "failed to marshal profile pb to string for profile %s: %s", profile.GetName(), err.Error())
	}

	req, err := http.NewRequest("POST", baseURL+"/v1/matchfunction:run", strings.NewReader(strReq))
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "failed to create mmf http request for profile %s: %s", profile.GetName(), err.Error())
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get response from mmf run for profile %s: %s", profile.Name, err.Error())
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			logger.WithError(err).Warning("failed to close response body read closer")
		}
	}()

	dec := json.NewDecoder(resp.Body)
	for {
		var item struct {
			Result json.RawMessage        `json:"result"`
			Error  map[string]interface{} `json:"error"`
		}

		err := dec.Decode(&item)
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Unavailable, "failed to read response from HTTP JSON stream: %s", err.Error())
		}
		if len(item.Error) != 0 {
			return status.Errorf(codes.Unavailable, "failed to execute matchfunction.Run: %v", item.Error)
		}
		resp := &pb.RunResponse{}
		if err := jsonpb.UnmarshalString(string(item.Result), resp); err != nil {
			return status.Errorf(codes.Unavailable, "failed to execute json.Unmarshal(%s, &resp): %v", item.Result, err)
		}
		select {
		case proposals <- resp.GetProposal():
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (s *backendService) ReleaseTickets(ctx context.Context, req *pb.ReleaseTicketsRequest) (*pb.ReleaseTicketsResponse, error) {
	err := s.store.DeleteTicketsFromPendingRelease(ctx, req.GetTicketIds())
	if err != nil {
		err = errors.Wrap(err, "failed to remove the awaiting tickets from the pending release for requested tickets")
		return nil, err
	}

	stats.Record(ctx, ticketsReleased.M(int64(len(req.TicketIds))))
	return &pb.ReleaseTicketsResponse{}, nil
}

func (s *backendService) ReleaseAllTickets(ctx context.Context, req *pb.ReleaseAllTicketsRequest) (*pb.ReleaseAllTicketsResponse, error) {
	err := s.store.ReleaseAllTickets(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.ReleaseAllTicketsResponse{}, nil
}

// AssignTickets overwrites the Assignment field of the input TicketIds.
func (s *backendService) AssignTickets(ctx context.Context, req *pb.AssignTicketsRequest) (*pb.AssignTicketsResponse, error) {
	resp, err := doAssignTickets(ctx, req, s.store)
	if err != nil {
		return nil, err
	}

	numIds := 0
	for _, ag := range req.Assignments {
		numIds += len(ag.TicketIds)
	}

	stats.Record(ctx, ticketsAssigned.M(int64(numIds)))
	return resp, nil
}

func createOrUpdateBackfill(ctx context.Context, match *pb.Match, store statestore.Service) error {
	backfill := match.GetBackfill()
	if backfill == nil {
		return nil
	}

	ticketIds := make([]string, len(match.Tickets))

	for _, t := range match.Tickets {
		ticketIds = append(ticketIds, t.Id)
	}

	if backfill.Id == "" {
		backfill.Id = xid.New().String()
		backfill.CreateTime = ptypes.TimestampNow()
		return store.CreateBackfill(ctx, backfill, ticketIds)
	}

	m := store.NewMutex(backfill.Id)
	err := m.Lock(ctx)
	if err != nil {
		return err
	}

	defer func() {
		_, unlockErr := m.Unlock(ctx)
		if unlockErr != nil {
			logger.WithFields(logrus.Fields{"backfill_id": backfill.Id}).WithError(unlockErr).Error("failed to make unlock")
		}
	}()

	bf, ids, err := store.GetBackfill(ctx, backfill.Id)
	if err != nil {
		return err
	}

	if bf.Generation != backfill.Generation {
		logger.WithFields(logrus.Fields{"backfill_id": backfill.Id}).
			WithError(errBackfillGenerationMismatch).
			Errorf("failed to update backfill, expecting: %d generation but got: %d", bf.Generation, backfill.Generation)
		return errBackfillGenerationMismatch
	}

	bf.SearchFields = backfill.SearchFields
	bf.Extensions = backfill.Extensions
	bf.Generation++

	return store.UpdateBackfill(ctx, bf, append(ids, ticketIds...))
}

func doAssignTickets(ctx context.Context, req *pb.AssignTicketsRequest, store statestore.Service) (*pb.AssignTicketsResponse, error) {
	resp, tickets, err := store.UpdateAssignments(ctx, req)
	if err != nil {
		return nil, err
	}

	for _, ticket := range tickets {
		err = recordTimeToAssignment(ctx, ticket)
		if err != nil {
			logger.WithError(err).Errorf("failed to record time to assignment for ticket %s", ticket.Id)
		}
	}

	ids := []string{}

	for _, ag := range req.Assignments {
		ids = append(ids, ag.TicketIds...)
	}

	for _, id := range ids {
		err = store.DeindexTicket(ctx, id)
		// Try to deindex all input tickets. Log without returning an error if the deindexing operation failed.
		// TODO: consider retry the index operation
		if err != nil {
			logger.WithError(err).Errorf("failed to deindex ticket %s after updating the assignments", id)
		}
	}

	if err = store.DeleteTicketsFromPendingRelease(ctx, ids); err != nil {
		logger.WithFields(logrus.Fields{
			"ticket_ids": ids,
		}).Error(err)
	}

	return resp, nil
}

func recordTimeToAssignment(ctx context.Context, ticket *pb.Ticket) error {
	if ticket.Assignment == nil {
		return fmt.Errorf("assignment for ticket %s is nil", ticket.Id)
	}

	now := time.Now()
	created, err := ptypes.Timestamp(ticket.CreateTime)
	if err != nil {
		return err
	}

	stats.Record(ctx, ticketsTimeToAssignment.M(now.Sub(created).Milliseconds()))

	return nil
}

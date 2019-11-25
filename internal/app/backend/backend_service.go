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
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/internal/statestore"
	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

// The service implementing the Backend API that is called to generate matches
// and make assignments for Tickets.
type backendService struct {
	synchronizer *synchronizerClient
	store        statestore.Service
	mmfClients   *rpc.ClientCache
}

type mmfResult struct {
	matches []*pb.Match
	err     error
}

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.backend",
	})
	mMatchesFetched  = telemetry.Counter("backend/matches_fetched", "matches fetched")
	mTicketsAssigned = telemetry.Counter("backend/tickets_assigned", "tickets assigned")

	mRegisterPhase           = telemetry.HistogramWithBounds("backend/register", "time to register and get synchronizer ID", "ms", telemetry.HistogramBounds)
	mProposalCollectionPhase = telemetry.HistogramWithBounds("backend/proposal_collection", "time to collect the proposals", "ms", telemetry.HistogramBounds)
	mEvaluateProposalsPhase  = telemetry.HistogramWithBounds("backend/evaluate_proposals", "time to evaluate proposals", "ms", telemetry.HistogramBounds)
)

// FetchMatches triggers a MatchFunction with the specified MatchProfiles, while each MatchProfile
// returns a set of match proposals. FetchMatches method streams the results back to the caller.
// FetchMatches immediately returns an error if it encounters any execution failures.
//   - If the synchronizer is enabled, FetchMatch will then call the synchronizer to deduplicate proposals with overlapped tickets.
func (s *backendService) FetchMatches(req *pb.FetchMatchesRequest, stream pb.Backend_FetchMatchesServer) error {
	if req.GetConfig() == nil {
		return status.Error(codes.InvalidArgument, ".config is required")
	}
	if req.GetProfiles() == nil {
		return status.Error(codes.InvalidArgument, ".profile is required")
	}

	resultChan := make(chan mmfResult, len(req.GetProfiles()))

	var syncID string
	var err error

	startTime := time.Now()

	syncID, err = s.synchronizer.register(stream.Context())
	if err != nil {
		return err
	}

	telemetry.RecordNUnitMeasurement(stream.Context(), mRegisterPhase, time.Since(startTime).Milliseconds())
	startTime = time.Now()

	err = doFetchMatchesReceiveMmfResult(stream.Context(), s.mmfClients, req, resultChan)
	if err != nil {
		return err
	}

	proposals, err := doFetchMatchesValidateProposals(stream.Context(), resultChan, len(req.GetProfiles()))
	if err != nil {
		return err
	}

	telemetry.RecordNUnitMeasurement(stream.Context(), mProposalCollectionPhase, time.Since(startTime).Milliseconds())
	startTime = time.Now()

	results, err := s.synchronizer.evaluate(stream.Context(), syncID, proposals)
	if err != nil {
		return err
	}

	telemetry.RecordNUnitMeasurement(stream.Context(), mEvaluateProposalsPhase, time.Since(startTime).Milliseconds())

	for _, result := range results {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
			err = stream.Send(&pb.FetchMatchesResponse{Match: result})
			telemetry.RecordUnitMeasurement(stream.Context(), mMatchesFetched)
			if err != nil {
				logger.WithError(err).Error("failed to stream back the response")
				return err
			}
		}
	}

	return nil
}

func doFetchMatchesReceiveMmfResult(ctx context.Context, mmfClients *rpc.ClientCache, req *pb.FetchMatchesRequest, resultChan chan<- mmfResult) error {
	var grpcClient pb.MatchFunctionClient
	var httpClient *http.Client
	var baseURL string
	var err error

	configType := req.GetConfig().GetType()
	address := fmt.Sprintf("%s:%d", req.GetConfig().GetHost(), req.GetConfig().GetPort())

	switch configType {
	// MatchFunction Hosted as a GRPC service
	case pb.FunctionConfig_GRPC:
		var conn *grpc.ClientConn
		conn, err = mmfClients.GetGRPC(address)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error":    err.Error(),
				"function": req.GetConfig(),
			}).Error("failed to establish grpc client connection to match function")
			return status.Error(codes.InvalidArgument, "failed to connect to match function")
		}
		grpcClient = pb.NewMatchFunctionClient(conn)
	// MatchFunction Hosted as a REST service
	case pb.FunctionConfig_REST:
		httpClient, baseURL, err = mmfClients.GetHTTP(address)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error":    err.Error(),
				"function": req.GetConfig(),
			}).Error("failed to establish rest client connection to match function")
			return status.Error(codes.InvalidArgument, "failed to connect to match function")
		}
	default:
		return status.Error(codes.InvalidArgument, "provided match function type is not supported")
	}

	for _, profile := range req.GetProfiles() {
		go func(profile *pb.MatchProfile) {
			// Get the match results that will be sent.
			// TODO: The matches returned by the MatchFunction will be sent to the
			// Evaluator to select results. Until the evaluator is implemented,
			// we channel all matches as accepted results.
			switch configType {
			case pb.FunctionConfig_GRPC:
				matches, err := matchesFromGRPCMMF(ctx, profile, grpcClient)
				resultChan <- mmfResult{matches, err}
			case pb.FunctionConfig_REST:
				matches, err := matchesFromHTTPMMF(ctx, profile, httpClient, baseURL)
				resultChan <- mmfResult{matches, err}
			}
		}(profile)
	}

	return nil
}

func doFetchMatchesValidateProposals(ctx context.Context, resultChan <-chan mmfResult, channelSize int) ([]*pb.Match, error) {
	proposals := []*pb.Match{}
	for i := 0; i < channelSize; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-resultChan:
			// Check if mmf responds with any errors
			if result.err != nil {
				return nil, result.err
			}

			// Check if mmf returns a match with no tickets in it
			for _, match := range result.matches {
				if len(match.GetTickets()) == 0 {
					return nil, status.Errorf(codes.FailedPrecondition, "match %s does not have associated tickets.", match.GetMatchId())
				}
			}
			proposals = append(proposals, result.matches...)
		}
	}
	return proposals, nil
}

func matchesFromHTTPMMF(ctx context.Context, profile *pb.MatchProfile, client *http.Client, baseURL string) ([]*pb.Match, error) {
	var m jsonpb.Marshaler
	strReq, err := m.MarshalToString(&pb.RunRequest{Profile: profile})
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed to marshal profile pb to string for profile %s: %s", profile.GetName(), err.Error())
	}

	req, err := http.NewRequest("POST", baseURL+"/v1/matchfunction:run", strings.NewReader(strReq))
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed to create mmf http request for profile %s: %s", profile.GetName(), err.Error())
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get response from mmf run for proile %s: %s", profile.Name, err.Error())
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			logger.WithError(err).Warning("failed to close response body read closer")
		}
	}()

	dec := json.NewDecoder(resp.Body)
	proposals := make([]*pb.Match, 0)
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
			return nil, status.Errorf(codes.Unavailable, "failed to read response from HTTP JSON stream: %s", err.Error())
		}
		if len(item.Error) != 0 {
			return nil, status.Errorf(codes.Unavailable, "failed to execute matchfunction.Run: %v", item.Error)
		}
		resp := &pb.RunResponse{}
		if err := jsonpb.UnmarshalString(string(item.Result), resp); err != nil {
			return nil, status.Errorf(codes.Unavailable, "failed to execute json.Unmarshal(%s, &resp): %v.", item.Result, err)
		}
		proposals = append(proposals, resp.GetProposal())
	}

	return proposals, nil
}

// matchesFromGRPCMMF triggers execution of MMFs to fetch match results for each profile.
// These proposals are then sent to evaluator and the results are streamed back on the channel
// that this function returns to the caller.
func matchesFromGRPCMMF(ctx context.Context, profile *pb.MatchProfile, client pb.MatchFunctionClient) ([]*pb.Match, error) {
	// TODO: This code calls user code and could hang. We need to add a deadline here
	// and timeout gracefully to ensure that the ListMatches completes.
	stream, err := client.Run(ctx, &pb.RunRequest{Profile: profile})
	if err != nil {
		logger.WithError(err).Error("failed to run match function for profile")
		return nil, err
	}

	proposals := make([]*pb.Match, 0)
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Errorf("%v.Run() error, %v\n", client, err)
			return nil, err
		}
		proposals = append(proposals, resp.GetProposal())
	}

	return proposals, nil
}

// AssignTickets overwrites the Assignment field of the input TicketIds.
func (s *backendService) AssignTickets(ctx context.Context, req *pb.AssignTicketsRequest) (*pb.AssignTicketsResponse, error) {
	err := doAssignTickets(ctx, req, s.store)
	if err != nil {
		logger.WithError(err).Error("failed to update assignments for requested tickets")
		return nil, err
	}

	telemetry.RecordNUnitMeasurement(ctx, mTicketsAssigned, int64(len(req.TicketIds)))
	return &pb.AssignTicketsResponse{}, nil
}

func doAssignTickets(ctx context.Context, req *pb.AssignTicketsRequest, store statestore.Service) error {
	err := store.UpdateAssignments(ctx, req.GetTicketIds(), req.GetAssignment())
	if err != nil {
		logger.WithError(err).Error("failed to update assignments")
		return err
	}
	for _, id := range req.GetTicketIds() {
		err = store.DeindexTicket(ctx, id)
		// Try to deindex all input tickets. Log without returning an error if the deindexing operation failed.
		// TODO: consider retry the index operation
		if err != nil {
			logger.WithError(err).Errorf("failed to deindex ticket %s after updating the assignments", id)
		}
	}

	if err = store.DeleteTicketsFromIgnoreList(ctx, req.GetTicketIds()); err != nil {
		logger.WithFields(logrus.Fields{
			"ticket_ids": req.GetTicketIds(),
		}).Error(err)
	}

	return nil
}

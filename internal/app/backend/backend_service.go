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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/internal/statestore"
)

// The service implementing the Backend API that is called to generate matches
// and make assignments for Tickets.
type backendService struct {
	cfg        config.View
	store      statestore.Service
	mmfClients mmfClients
}

type mmfClients struct {
	addrMap    sync.Map
	evalClient pb.EvaluatorClient
}

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.backend.backend_service",
	})
)

// newBackend creates and initializes the backend service.
func newBackend(cfg config.View) (*backendService, error) {
	bs := &backendService{
		cfg: cfg,
	}

	// Initialize the state storage interface.
	var err error
	bs.store, err = statestore.New(cfg)
	if err != nil {
		return nil, err
	}

	return bs, nil
}

// FetchMatches triggers execution of the specfied MatchFunction for each of the
// specified MatchProfiles. Each MatchFunction execution returns a set of
// proposals which are then evaluated to generate results. FetchMatches method
// streams these results back to the caller.
func (s *backendService) FetchMatches(req *pb.FetchMatchesRequest, stream pb.Backend_FetchMatchesServer) error {
	// Validate that the function configuration and match profiles are provided.
	if req.Config == nil {
		logger.Error("invalid argument, match function config is nil")
		return status.Errorf(codes.InvalidArgument, "match function configuration needs to be provided")
	}

	var grpcClient pb.MatchFunctionClient
	var httpClient *http.Client
	var baseURL string
	var err error

	switch (req.Config.Type).(type) {
	// MatchFunction Hosted as a GRPC service
	case *pb.FunctionConfig_Grpc:
		grpcClient, err = s.getGRPCClient((req.Config.Type).(*pb.FunctionConfig_Grpc))
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error":    err.Error(),
				"function": req.Config,
			}).Error("failed to establish grpc client connection to match function")
			return status.Error(codes.InvalidArgument, "failed to connect to match function")
		}
	// MatchFunction Hosted as a REST service
	case *pb.FunctionConfig_Rest:
		httpClient, baseURL, err = s.getHTTPClient((req.Config.Type).(*pb.FunctionConfig_Rest))
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error":    err.Error(),
				"function": req.Config,
			}).Error("failed to establish rest client connection to match function")
			return status.Error(codes.InvalidArgument, "failed to connect to match function")
		}
	default:
		logger.Error("unsupported function type provided")
		return status.Error(codes.InvalidArgument, "provided match function type is not supported")
	}

	ctx := stream.Context()
	matchChan := make(chan *pb.Match)
	errChan := make(chan error)

	go func(matchChan chan<- *pb.Match, errChan chan<- error) {
		var wg sync.WaitGroup

		defer func() {
			wg.Wait()
			close(matchChan)
			close(errChan)
		}()

		for _, profile := range req.Profile {
			wg.Add(1)
			go func(profile *pb.MatchProfile) {
				defer wg.Done()

				// Get the channel over which the generated match results will be sent.
				switch (req.Config.Type).(type) {
				case *pb.FunctionConfig_Grpc:
					matchesFromGRPCMMF(ctx, profile, grpcClient, matchChan, errChan)
				case *pb.FunctionConfig_Rest:
					matchesFromHTTPMMF(ctx, profile, httpClient, baseURL, matchChan, errChan)
				}
			}(profile)
		}
	}(matchChan, errChan)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errChan:
			return err
		case match, ok := <-matchChan:
			if !ok {
				// Channel closed indicating that we are done processing matches.
				return nil
			}

			// Send the match result over output channel.
			m, ok := proto.Clone(match).(*pb.Match)
			if !ok {
				logger.Error("failed to clone match proto")
				return status.Error(codes.Internal, "failed processing match result")
			}

			err := stream.Send(&pb.FetchMatchesResponse{Match: m})
			if err != nil {
				return err
			}
		}
	}
}

func (s *backendService) getHTTPClient(config *pb.FunctionConfig_Rest) (*http.Client, string, error) {
	client, baseURL, err := rpc.HTTPClientFromEndpoint(s.cfg, config.Rest.Host, int(config.Rest.Port))
	if err != nil {
		return nil, "", err
	}
	return client, baseURL, err
}

func (s *backendService) getGRPCClient(config *pb.FunctionConfig_Grpc) (pb.MatchFunctionClient, error) {
	addr := fmt.Sprintf("%s:%d", config.Grpc.Host, config.Grpc.Port)
	client, ok := s.mmfClients.addrMap.Load(addr)
	if !ok {
		conn, err := rpc.GRPCClientFromEndpoint(s.cfg, config.Grpc.Host, int(config.Grpc.Port))
		if err != nil {
			return nil, err
		}

		client = pb.NewMatchFunctionClient(conn)
		s.mmfClients.addrMap.Store(addr, client)
		logger.WithFields(logrus.Fields{
			"address": config,
		}).Info("successfully connected to the GRPC match function service")
	}

	return client.(pb.MatchFunctionClient), nil
}

func matchesFromHTTPMMF(ctx context.Context, profile *pb.MatchProfile, client *http.Client, baseURL string, matchChan chan<- *pb.Match, errChan chan<- error) {
	jsonProfile, err := json.Marshal(profile)
	if err != nil {
		errChan <- status.Error(codes.FailedPrecondition, err.Error())
		logger.WithError(err).Error("failed to marshal profile pb to string")
		return
	}

	reqBody, err := json.Marshal(map[string]json.RawMessage{"profile": jsonProfile})
	if err != nil {
		errChan <- status.Error(codes.FailedPrecondition, err.Error())
		logger.WithError(err).Error("failed to marshal request body")
		return
	}

	req, err := http.NewRequest("POST", baseURL+"/v1/matchfunction:run", bytes.NewBuffer(reqBody))
	if err != nil {
		errChan <- status.Error(codes.FailedPrecondition, err.Error())
		logger.WithError(err).Error("failed to create mmf http request")
		return
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		errChan <- status.Errorf(codes.Internal, "failed to get response from mmf run: %s", err.Error())
		return
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			logger.WithError(err).Error("failed to close response body read closer")
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errChan <- status.Error(codes.FailedPrecondition, err.Error())
		logger.WithError(err).Error("failed to read from response body")
		return
	}

	pbResp := &pb.RunResponse{}
	err = json.Unmarshal(body, pbResp)
	if err != nil {
		errChan <- status.Error(codes.FailedPrecondition, err.Error())
		logger.WithError(err).Error("failed to unmarshal response body to response pb")
		return
	}

	for _, match := range pbResp.Proposal {
		matchChan <- match
	}
}

// matchesFromGRPCMMF triggers execution of MMFs to fetch match results for each profile.
// These proposals are then sent to evaluator and the results are streamed back on the channel
// that this function returns to the caller.
func matchesFromGRPCMMF(ctx context.Context, profile *pb.MatchProfile, client pb.MatchFunctionClient, matchChan chan<- *pb.Match, errChan chan<- error) {
	// TODO: This code calls user code and could hang. We need to add a deadline here
	// and timeout gracefully to ensure that the ListMatches completes.
	// TODO: Currently, a failure in running the MMF is silently ignored and does not
	// fail the FetchMatches. This needs to be revisited to investigate whether the
	// FetchMatches should fail or atleast hint that a part of MMF executions failed.
	resp, err := client.Run(ctx, &pb.RunRequest{Profile: profile})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":   err.Error(),
			"profile": profile,
		}).Error("failed to run match function for profile")
		errChan <- err
		return
	}

	logger.WithFields(logrus.Fields{
		"profile":   profile,
		"proposals": resp.Proposal,
	}).Trace("proposals generated for match profile")
	// TODO: The matches returned by the MatchFunction will be sent to the
	// Evaluator to select results. Until the evaluator is implemented,
	// we channel all matches as accepted results.
	for _, match := range resp.Proposal {
		matchChan <- match
	}
}

// AssignTickets sets the specified Assignment on the Tickets for the Ticket
// ids passed.
func (s *backendService) AssignTickets(ctx context.Context, req *pb.AssignTicketsRequest) (*pb.AssignTicketsResponse, error) {
	err := s.store.UpdateAssignments(ctx, req.TicketId, req.Assignment)
	if err != nil {
		logger.WithError(err).Error("failed to update assignments for requested tickets")
		return nil, err
	}

	return &pb.AssignTicketsResponse{}, nil
}

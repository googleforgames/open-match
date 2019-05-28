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
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/statestore"
)

// The service implementing the Backend API that is called to generate matches
// and make assignments for Tickets.
type backendService struct {
	cfg        config.View
	store      statestore.Service
	mmfClients sync.Map
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

	ctx := stream.Context()
	var c <-chan *pb.Match
	switch (req.Config.Type).(type) {
	// MatchFunction Hosted as a GRPC service
	case *pb.FunctionConfig_Grpc:
		client, err := s.getGRPCClient((req.Config.Type).(*pb.FunctionConfig_Grpc))
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error":    err.Error(),
				"function": req.Config,
			}).Error("failed to connect to match function")
			return status.Error(codes.InvalidArgument, "failed to connect to match function")
		}

		// Get the channel over which the generated match results will be sent.
		c = s.matchesFromGrpcMMF(ctx, req.Profile, client)

	// MatchFunction Hosted as a REST service
	case *pb.FunctionConfig_Rest:
		return status.Error(codes.Unimplemented, "not implemented")

	default:
		logger.Error("unsupported function type provided")
		return status.Error(codes.InvalidArgument, "provided match function type is not supported")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case match, ok := <-c:
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

func (s *backendService) getGRPCClient(config *pb.FunctionConfig_Grpc) (pb.MatchFunctionClient, error) {
	addr := fmt.Sprintf("%s:%d", config.Grpc.Host, config.Grpc.Port)
	client, ok := s.mmfClients.Load(addr)
	if !ok {
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			return nil, err
		}

		client = pb.NewMatchFunctionClient(conn)
		s.mmfClients.Store(addr, client)
		logger.WithFields(logrus.Fields{
			"address": config,
		}).Info("successfully connected to the GRPC match function service")
	}

	return client.(pb.MatchFunctionClient), nil
}

// matchesFromGrpcMMF triggers execution of MMFs to fetch match results for each profile.
// These proposals are then sent to evaluator and the results are streamed back on the channel
// that this function returns to the caller.
func (s *backendService) matchesFromGrpcMMF(ctx context.Context, profiles []*pb.MatchProfile, client pb.MatchFunctionClient) <-chan *pb.Match {
	c := make(chan *pb.Match)

	// Create a goroutine that will wait for completion of individual goroutines processing each
	// match profile and then close the channel signaling completion of match results generation.
	go func(c chan<- *pb.Match) {
		var wg sync.WaitGroup
		for _, profile := range profiles {
			// Trigger a goroutine for each profile. This will call the MatchFunction to get
			// proposed matches and will then pass these to evaluator to get the accepted
			// match results that it will send on the output channel.
			wg.Add(1)
			go func(profile *pb.MatchProfile) {
				defer wg.Done()
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
					c <- match
				}
			}(profile)
		}

		// Wait for all the goroutines handling each profile to return and then close the channel.
		wg.Wait()
		close(c)
	}(c)

	return c
}

// AssignTickets sets the specified Assignment on the Tickets for the Ticket
// ids passed.
func (s *backendService) AssignTickets(ctx context.Context, req *pb.AssignTicketsRequest) (*pb.AssignTicketsResponse, error) {
	return &pb.AssignTicketsResponse{}, nil
}

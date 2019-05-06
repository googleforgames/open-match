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

	"open-match.dev/open-match/internal/future/pb"
)

// The service implementing the Backent API that is called to generate matches
// and make assignments for Tickets.
type backendService struct {
}

// GetMatches triggers execution of the specfied MatchFunction for each of the
// specified MatchProfiles. Each MatchFunction execution returns a set of
// proposals which are then evaluated to generate results. GetMatches method
// streams these results back to the caller.
// TODO: Should this be renamed to createProposal? It's not a "Get" if it's
// executing a MatchFunction.
func (s *backendService) GetMatches(req *pb.GetMatchesRequest, stream pb.Backend_GetMatchesServer) error {
	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			return nil

		default:
			err := stream.Send(&pb.GetMatchesResponse{})
			if err != nil {
				return err
			}
		}
	}
}

// AssignTickets sets the specified Assignment on the Tickets for the Ticket
// ids passed.
func (s *backendService) AssignTickets(ctx context.Context, req *pb.AssignTicketsRequest) (*pb.AssignTicketsResponse, error) {
	return &pb.AssignTicketsResponse{}, nil
}

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

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"google.golang.org/grpc"
	"open-match.dev/open-match/pkg/pb"
)

// The Director in this tutorial continously polls Open Match for the Match
// Profiles and makes random assignments for the Tickets in the returned matches.

const (
	// The endpoint for the Open Match Backend service.
	omBackendEndpoint = "open-match-backend.open-match.svc.cluster.local:50505"
	// The Host and Port for the Match Function service endpoint.
	functionHostName       = "backfill-matchfunction.backfill.svc.cluster.local"
	functionPort     int32 = 50502
)

var (
	be                 pb.BackendServiceClient
	scenarioWaitLobby  chan int
	scenarioFillVacant chan int
)

func init() {
	scenarioWaitLobby = make(chan int, 1)
	scenarioFillVacant = make(chan int, 1)
	scenarioWaitLobby <- 1
}

func main() {
	// Connect to Open Match Backend.
	conn, err := grpc.Dial(omBackendEndpoint, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to Open Match Backend, got %s", err.Error())
	}

	defer conn.Close()
	be = pb.NewBackendServiceClient(conn)

	for {
		select {
		case <-scenarioWaitLobby:
			waitLobby()
		case <-scenarioFillVacant:

		}
	}
}

func waitLobby() {
	var noResultCount int
	for range time.Tick(time.Second * 5) {
		matches, err := fetch()
		if err != nil {
			log.Printf("Failed to fetch matches for profile, got %s", err.Error())
			return
		}
		if len(matches) == 0 {
			noResultCount += 1
		}
		log.Printf("Generated %v matches for profile", len(matches))
		if err := assign(be, matches); err != nil {
			log.Printf("Failed to assign servers to matches, got %s", err.Error())
			return
		}
		if noResultCount > 20 {
			break
		}
	}
}

func fetch() ([]*pb.Match, error) {
	req := &pb.FetchMatchesRequest{
		Config: &pb.FunctionConfig{
			Host: functionHostName,
			Port: functionPort,
			Type: pb.FunctionConfig_GRPC,
		},
		Profile: &pb.MatchProfile{
			Name: "1v1",
			Pools: []*pb.Pool{
				{
					Name: "Everyone",
				},
			},
		},
	}

	stream, err := be.FetchMatches(context.Background(), req)
	if err != nil {
		log.Println()
		return nil, err
	}

	var result []*pb.Match
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		result = append(result, resp.GetMatch())
	}

	return result, nil
}

func assign(be pb.BackendServiceClient, matches []*pb.Match) error {
	for _, match := range matches {
		ticketIDs := []string{}
		for _, t := range match.GetTickets() {
			ticketIDs = append(ticketIDs, t.Id)
		}

		conn := fmt.Sprintf("%d.%d.%d.%d:2222", 127, 0, 0, 1)
		req := &pb.AssignTicketsRequest{
			Assignments: []*pb.AssignmentGroup{
				{
					TicketIds: ticketIDs,
					Assignment: &pb.Assignment{
						Connection: conn,
					},
				},
			},
		}

		if _, err := be.AssignTickets(context.Background(), req); err != nil {
			return fmt.Errorf("AssignTickets failed for match %v, got %w", match.GetMatchId(), err)
		}

		log.Printf("Assigned server %v to match %v", conn, match.GetMatchId())
	}

	return nil
}

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
	"math/rand"
	"sync"
	"time"

	"google.golang.org/grpc"
	"open-match.dev/open-match/pkg/pb"
)

// The Director in this tutorial does the following:
// - Generates the profiles that it will poll for.
// - Polls Open Match backend for those profiles.
//
// The Director can be configured to make a single fetch for all the profiles
// or can keep polling forever at a configured interval.

var (
	omBackendEndpoint       = "om-backend.open-match.svc.cluster.local:50505"
	functionHostName        = "mm101-tutorial-matchfunction.mm101-tutorial.svc.cluster.local"
	functionPort      int32 = 50502
	interval                = 5000 // Millisecond duration for the polling interval.
	fetchForever            = true // Fetch once if false, forever if true.
)

func main() {
	// Connect to Open Match Backend.
	conn, err := grpc.Dial(omBackendEndpoint, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to Open Match Backend, got %w", err)
	}

	defer conn.Close()
	be := pb.NewBackendClient(conn)

	// Generate the profiles to fetch matches for.
	profiles := generateProfiles()
	log.Printf("Fetching matches for %v profiles", len(profiles))

	// Fetch matches for each profile and then wait for all the calls to return
	// before starting the next iteration.
	for {
		var wg sync.WaitGroup
		for _, p := range profiles {
			wg.Add(1)
			go func(wg *sync.WaitGroup, p *pb.MatchProfile) {
				defer wg.Done()
				matches, err := fetch(be, p)
				if err != nil {
					log.Printf("Failed to fetch matches for profile %v, got %w", p.GetName(), err)
					return
				}

				log.Printf("Generated %v matches for profile %v", len(matches), p.GetName())

				// Make random assignment for all the matches returned for this profile.
				if err := assign(be, matches); err != nil {
					log.Printf("Failed to assign servers to matches, got %w", err)
					return
				}
			}(&wg, p)
		}

		wg.Wait()
		if !fetchForever {
			break
		}
		time.Sleep(time.Millisecond * time.Duration(interval))
	}

	// Block forever to enable inspecting state.
	select {}
}

func fetch(be pb.BackendClient, p *pb.MatchProfile) ([]*pb.Match, error) {
	req := &pb.FetchMatchesRequest{
		Config: &pb.FunctionConfig{
			Host: functionHostName,
			Port: functionPort,
			Type: pb.FunctionConfig_GRPC,
		},
		Profiles: []*pb.MatchProfile{p},
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

func assign(be pb.BackendClient, matches []*pb.Match) error {
	for _, match := range matches {
		ticketIDs := []string{}
		for _, t := range match.GetTickets() {
			ticketIDs = append(ticketIDs, t.Id)
		}

		conn := fmt.Sprintf("%d.%d.%d.%d:2222", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256))
		req := &pb.AssignTicketsRequest{
			TicketIds: ticketIDs,
			Assignment: &pb.Assignment{
				Connection: conn,
			},
		}

		if _, err := be.AssignTickets(context.Background(), req); err != nil {
			return fmt.Errorf("AssignTickets failed for match %v, got %w", match.GetMatchId(), err)
		}

		log.Printf("Assigned server %v to match %v", conn, match.GetMatchId())
	}

	return nil
}

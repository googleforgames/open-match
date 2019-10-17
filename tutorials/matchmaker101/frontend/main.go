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

// The Frontend in this tutorial creates a configured number of Tickets in Open
// Match per iteration. The Frontend can be configured to run for a single
// iteration or to keep running iterations forever at a configured interval.

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"open-match.dev/open-match/pkg/pb"
)

var (
	omBackendEndpoint = "om-frontend.open-match.svc.cluster.local:50504"
	ticketsPerIter    = 20   // Number of tickets created per iteration
	interval          = 1000 // Millisecond interval between each iteration.
	createForever     = true // Iterate once if false, forever if true.
)

func main() {
	// Connect to Open Match Frontend.
	conn, err := grpc.Dial(omBackendEndpoint, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to Open Match, got %v", err)
	}

	defer conn.Close()
	fe := pb.NewFrontendClient(conn)

	for {
		for i := 0; i <= ticketsPerIter; i++ {
			req := &pb.CreateTicketRequest{
				Ticket: makeTicket(),
			}

			resp, err := fe.CreateTicket(context.Background(), req)
			if err != nil {
				log.Fatalf("Failed to Create Ticket, got %w", err)
			}

			log.Println("Ticket created successfully, id:", resp.Ticket.Id)
			go deleteOnAssign(fe, resp.Ticket)
		}

		if !createForever {
			break
		}

		time.Sleep(time.Millisecond * time.Duration(interval))
	}

	// Block forever to enable inspecting state.
	select {}
}

// deleteOnAssign fetches the Ticket state periodically and deletes the Ticket
// once it has an assignment.
func deleteOnAssign(fe pb.FrontendClient, t *pb.Ticket) {
	for {
		got, err := fe.GetTicket(context.Background(), &pb.GetTicketRequest{TicketId: t.GetId()})
		if err != nil {
			log.Fatalf("Failed to Get Ticket %v, got %w", t.GetId(), err)
		}

		if got.GetAssignment() != nil {
			log.Printf("Ticket %v got assignment %v", got.GetId(), got.GetAssignment())
			break
		}

		time.Sleep(time.Second * 1)
	}

	_, err := fe.DeleteTicket(context.Background(), &pb.DeleteTicketRequest{TicketId: t.GetId()})
	if err != nil {
		log.Fatalf("Failed to Delete Ticket %v, got %w", t.GetId(), err)
	}
}
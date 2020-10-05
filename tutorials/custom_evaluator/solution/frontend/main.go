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

// The Frontend in this tutorial continously creates Tickets in batches in Open Match.

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"open-match.dev/open-match/pkg/pb"
)

const (
	// The endpoint for the Open Match Frontend service.
	omFrontendEndpoint = "open-match-frontend.open-match.svc.cluster.local:50504"
	// Number of tickets created per iteration
	ticketsPerIter = 20
)

func main() {
	// Connect to Open Match Frontend.
	conn, err := grpc.Dial(omFrontendEndpoint, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to Open Match, got %v", err)
	}

	defer conn.Close()
	fe := pb.NewFrontendServiceClient(conn)
	for range time.Tick(time.Second * 2) {
		for i := 0; i <= ticketsPerIter; i++ {
			req := &pb.CreateTicketRequest{
				Ticket: makeTicket(),
			}

			resp, err := fe.CreateTicket(context.Background(), req)
			if err != nil {
				log.Printf("Failed to Create Ticket, got %s", err.Error())
				continue
			}

			log.Println("Ticket created successfully, id:", resp.Id)
			go deleteOnAssign(fe, resp)
		}
	}
}

// deleteOnAssign fetches the Ticket state periodically and deletes the Ticket
// once it has an assignment.
func deleteOnAssign(fe pb.FrontendServiceClient, t *pb.Ticket) {
	for {
		got, err := fe.GetTicket(context.Background(), &pb.GetTicketRequest{TicketId: t.GetId()})
		if err != nil {
			log.Fatalf("Failed to Get Ticket %v, got %s", t.GetId(), err.Error())
		}

		if got.GetAssignment() != nil {
			log.Printf("Ticket %v got assignment %v", got.GetId(), got.GetAssignment())
			break
		}

		time.Sleep(time.Second * 1)
	}

	_, err := fe.DeleteTicket(context.Background(), &pb.DeleteTicketRequest{TicketId: t.GetId()})
	if err != nil {
		log.Fatalf("Failed to Delete Ticket %v, got %s", t.GetId(), err.Error())
	}
}

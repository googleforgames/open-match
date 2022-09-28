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
)

var (
	fe                 pb.FrontendServiceClient
	scenarioWaitLobby  chan int
	scenarioFillVacant chan int
)

func init() {
	scenarioWaitLobby = make(chan int, 1)
	scenarioFillVacant = make(chan int, 1)
	scenarioWaitLobby <- 1
}

func main() {
	// Connect to Open Match Frontend.
	conn, err := grpc.Dial(omFrontendEndpoint, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to Open Match, got %v", err)
	}

	defer conn.Close()
	fe = pb.NewFrontendServiceClient(conn)
	for {
		select {
		case <-scenarioWaitLobby:
			for i := 0; i < 5; i++ {
				go createPlayerWaitLobby(i)
				time.Sleep(3 * time.Second)
			}

		case <-scenarioFillVacant:

		}
	}
}

func createPlayerWaitLobby(i int) {
	defer handlePanic()
	log.Printf("Player %d: Started game app and ready to play", i)
	ticketRequest := &pb.CreateTicketRequest{
		Ticket: &pb.Ticket{},
	}

	ticketResponse, err := fe.CreateTicket(context.Background(), ticketRequest)
	if err != nil {
		log.Printf("Player %d: Failed to Create Ticket, got %s", i, err.Error())
		return
	}

	log.Println("Player ", i, ": Ticket created successfully, id:", ticketResponse.Id)

	assignmentRequest := &pb.WatchAssignmentsRequest{
		TicketId: ticketResponse.Id,
	}

	stream, err := fe.WatchAssignments(context.Background(), assignmentRequest)
	for {
		assignmentResponse, err := stream.Recv()
		if err != nil {
			// For now we don't expect to get EOF, so that's still an error worthy of panic.
			panic(err)
		}

		if assignmentResponse.Assignment.Connection != "" {
			log.Println("Player ", i, ": Ticket id:", ticketResponse.Id, "  got assignment --> ", assignmentResponse.Assignment.Connection)
			break
		}
	}

	err = stream.CloseSend()
	if err != nil {
		panic(err)
	}
	_, err = fe.DeleteTicket(context.Background(), &pb.DeleteTicketRequest{TicketId: ticketResponse.GetId()})
	if err != nil {
		log.Printf("Player %d: Failed to Delete Ticket %v, got %s", i, ticketResponse.Id, err.Error())
	}
	log.Printf("Player %d: Joined game server or waiting in lobby", i)
}

// recover function to handle panic
func handlePanic() {
	// detect if panic occurs or not
	a := recover()
	if a != nil {
		log.Println("RECOVER", a)
	}
}

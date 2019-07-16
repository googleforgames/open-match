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

package frontend

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/statestore"
	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

// frontendService implements the Frontend service that is used to create
// Tickets and add, remove them from the pool for matchmaking.
type frontendService struct {
	cfg   config.View
	store statestore.Service
}

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.frontend",
	})
	mTicketsCreated             = telemetry.Counter("frontend/tickets_created", "tickets created")
	mTicketsDeleted             = telemetry.Counter("frontend/tickets_deleted", "tickets deleted")
	mTicketsRetrieved           = telemetry.Counter("frontend/tickets_retrieved", "tickets retrieved")
	mTicketAssignmentsRetrieved = telemetry.Counter("frontend/tickets_assignments_retrieved", "ticket assignments retrieved")
)

// CreateTicket will create a new ticket, assign an id to it and put it in state
// storage. It will index the Ticket attributes based on the indexing configuration.
// Indexing a Ticket adds the it to the pool of Tickets considered for matchmaking.
func (s *frontendService) CreateTicket(ctx context.Context, req *pb.CreateTicketRequest) (*pb.CreateTicketResponse, error) {
	// Perform input validation.
	if req.GetTicket() == nil {
		return nil, status.Errorf(codes.InvalidArgument, ".ticket is required")
	}

	return doCreateTicket(ctx, req, s.store)
}

func doCreateTicket(ctx context.Context, req *pb.CreateTicketRequest, store statestore.Service) (*pb.CreateTicketResponse, error) {
	// Generate a ticket id and create a Ticket in state storage
	ticket, ok := proto.Clone(req.Ticket).(*pb.Ticket)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to clone input ticket proto")
	}

	ticket.Id = xid.New().String()
	err := store.CreateTicket(ctx, ticket)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":  err.Error(),
			"ticket": ticket,
		}).Error("failed to create the ticket")
		return nil, err
	}

	err = store.IndexTicket(ctx, ticket)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":  err.Error(),
			"ticket": ticket,
		}).Error("failed to index the ticket")
		return nil, err
	}

	telemetry.IncrementCounter(ctx, mTicketsCreated)
	return &pb.CreateTicketResponse{Ticket: ticket}, nil
}

// DeleteTicket removes the Ticket from state storage and from corresponding
// configured indices and lazily removes the ticket from state storage.
// Deleting a ticket immediately stops the ticket from being
// considered for future matchmaking requests, yet when the ticket itself will be deleted
// is undeterministic. Users may still be able to assign/get a ticket after calling DeleteTicket on it.
func (s *frontendService) DeleteTicket(ctx context.Context, req *pb.DeleteTicketRequest) (*pb.DeleteTicketResponse, error) {
	err := doDeleteTicket(ctx, req.GetTicketId(), s.store)
	if err != nil {
		return nil, err
	}
	telemetry.IncrementCounter(ctx, mTicketsDeleted)
	return &pb.DeleteTicketResponse{}, nil
}

func doDeleteTicket(ctx context.Context, id string, store statestore.Service) error {
	// Deindex this Ticket to remove it from matchmaking pool.
	err := store.DeindexTicket(ctx, id)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
			"id":    id,
		}).Error("failed to deindex the ticket")
		return err
	}

	//'lazy' ticket delete that should be called after a ticket
	// has been deindexed.
	go func() {
		err := store.DeleteTicket(context.Background(), id)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error": err.Error(),
				"id":    id,
			}).Error("failed to delete the ticket")
			return
		}
		// TODO: If other redis queues are implemented or we have custom index fields
		// created by Open Match, those need to be cleaned up here.
	}()
	return nil
}

// GetTicket returns the Ticket associated with the specified Ticket id.
func (s *frontendService) GetTicket(ctx context.Context, req *pb.GetTicketRequest) (*pb.Ticket, error) {
	telemetry.IncrementCounter(ctx, mTicketsRetrieved)
	return doGetTickets(ctx, req.GetTicketId(), s.store)
}

func doGetTickets(ctx context.Context, id string, store statestore.Service) (*pb.Ticket, error) {
	ticket, err := store.GetTicket(ctx, id)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
			"id":    id,
		}).Error("failed to get the ticket")
		return nil, err
	}

	return ticket, nil
}

// GetTicketUpdates streams matchmaking results from Open Match for the
// provided Ticket id.
func (s *frontendService) GetAssignments(req *pb.GetAssignmentsRequest, stream pb.Frontend_GetAssignmentsServer) error {
	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sender := func(assignment *pb.Assignment) error {
				telemetry.IncrementCounter(ctx, mTicketAssignmentsRetrieved)
				return stream.Send(&pb.GetAssignmentsResponse{Assignment: assignment})
			}
			return doGetAssignments(ctx, req.GetTicketId(), sender, s.store)
		}
	}
}

func doGetAssignments(ctx context.Context, id string, sender func(*pb.Assignment) error, store statestore.Service) error {
	var currAssignment *pb.Assignment
	var ok bool
	callback := func(assignment *pb.Assignment) error {
		if (currAssignment == nil && assignment != nil) || !proto.Equal(currAssignment, assignment) {
			currAssignment, ok = proto.Clone(assignment).(*pb.Assignment)
			if !ok {
				return status.Error(codes.Internal, "failed to cast the assignment object")
			}

			err := sender(currAssignment)
			if err != nil {
				logger.WithError(err).Error("failed to send Redis response to grpc server")
				return status.Errorf(codes.Aborted, err.Error())
			}
		}
		return nil
	}

	return store.GetAssignments(ctx, id, callback)
}

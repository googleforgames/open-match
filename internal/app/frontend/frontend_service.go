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
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/statestore"
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
		"component": "app.frontend.frontend_service",
	})
)

// newFrontend creates and initializes the frontend service.
func newFrontend(cfg config.View) (*frontendService, error) {
	fs := &frontendService{
		cfg: cfg,
	}

	// Initialize the state storage interface.
	var err error
	fs.store, err = statestore.New(cfg)
	if err != nil {
		return nil, err
	}

	return fs, nil
}

// CreateTicket will create a new ticket, assign an id to it and put it in state
// storage. It will index the Ticket attributes based on the indexing configuration.
// Indexing a Ticket adds the it to the pool of Tickets considered for matchmaking.
func (s *frontendService) CreateTicket(ctx context.Context, req *pb.CreateTicketRequest) (*pb.CreateTicketResponse, error) {
	// Perform input validation.
	if req.Ticket == nil {
		logger.Error("invalid argument - ticket cannot be nil")
		return nil, status.Errorf(codes.InvalidArgument, "ticket cannot be nil")
	}

	// Generate a ticket id and create a Ticket in state storage
	ticket, ok := proto.Clone(req.Ticket).(*pb.Ticket)
	if !ok {
		logger.Error("failed to clone input ticket proto")
		return nil, status.Error(codes.Internal, "failed processing input")
	}

	ticket.Id = xid.New().String()
	err := s.store.CreateTicket(ctx, ticket)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":  err.Error(),
			"ticket": ticket,
		}).Error("failed to create the ticket")
		return nil, err
	}

	err = s.store.IndexTicket(ctx, ticket)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":  err.Error(),
			"ticket": ticket,
		}).Error("failed to index the ticket")
		return nil, err
	}

	return &pb.CreateTicketResponse{Ticket: ticket}, nil
}

// DeleteTicket removes the Ticket from the configured indexes, thereby removing
// it from matchmaking pool. It also lazily removes the ticket from state storage.
func (s *frontendService) DeleteTicket(ctx context.Context, req *pb.DeleteTicketRequest) (*pb.DeleteTicketResponse, error) {
	// Deindex this Ticket to remove it from matchmaking pool.
	err := s.store.DeindexTicket(ctx, req.TicketId)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
			"id":    req.TicketId,
		}).Error("failed to deindex the ticket")
		return nil, err
	}

	// Kick off delete but don't wait for it to complete.
	go s.deleteTicket(req.TicketId)
	return &pb.DeleteTicketResponse{}, nil
}

// deleteTicket is a 'lazy' ticket delete that should be called after a ticket
// has been deindexed.
func (s *frontendService) deleteTicket(id string) {
	err := s.store.DeleteTicket(context.Background(), id)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
			"id":    id,
		}).Error("failed to delete the ticket")
		return
	}

	// TODO: If other redis queues are implemented or we have custom index fields
	// created by Open Match, those need to be cleaned up here.
}

// GetTicket returns the Ticket associated with the specified Ticket id.
func (s *frontendService) GetTicket(ctx context.Context, req *pb.GetTicketRequest) (*pb.Ticket, error) {
	ticket, err := s.store.GetTicket(ctx, req.TicketId)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
			"id":    req.TicketId,
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
			return nil
		default:
			callback := func(assignment *pb.Assignment) error {
				err := stream.Send(&pb.GetAssignmentsResponse{Assignment: assignment})
				if err != nil {
					logger.WithError(err).Error("Failed to send Redis response to grpc server")
					return status.Errorf(codes.Aborted, err.Error())
				}
				return nil
			}

			err := s.store.GetAssignments(ctx, req.TicketId, callback)
			if err != nil {
				return err
			}

			return nil
		}
	}
}

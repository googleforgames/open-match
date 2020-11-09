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

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/statestore"
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
)

// CreateTicket assigns an unique TicketId to the input Ticket and record it in state storage.
// A ticket is considered as ready for matchmaking once it is created.
//   - If a TicketId exists in a Ticket request, an auto-generated TicketId will override this field.
//   - If SearchFields exist in a Ticket, CreateTicket will also index these fields such that one can query the ticket with query.QueryTickets function.
func (s *frontendService) CreateTicket(ctx context.Context, req *pb.CreateTicketRequest) (*pb.Ticket, error) {
	// Perform input validation.
	if req.Ticket == nil {
		return nil, status.Errorf(codes.InvalidArgument, ".ticket is required")
	}
	if req.Ticket.Assignment != nil {
		return nil, status.Errorf(codes.InvalidArgument, "tickets cannot be created with an assignment")
	}
	if req.Ticket.CreateTime != nil {
		return nil, status.Errorf(codes.InvalidArgument, "tickets cannot be created with create time set")
	}

	return doCreateTicket(ctx, req, s.store)
}

func doCreateTicket(ctx context.Context, req *pb.CreateTicketRequest, store statestore.Service) (*pb.Ticket, error) {
	// Generate a ticket id and create a Ticket in state storage
	ticket, ok := proto.Clone(req.Ticket).(*pb.Ticket)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to clone input ticket proto")
	}

	ticket.Id = xid.New().String()
	ticket.CreateTime = ptypes.TimestampNow()

	sfCount := 0
	sfCount += len(ticket.GetSearchFields().GetDoubleArgs())
	sfCount += len(ticket.GetSearchFields().GetStringArgs())
	sfCount += len(ticket.GetSearchFields().GetTags())
	stats.Record(ctx, searchFieldsPerTicket.M(int64(sfCount)))
	stats.Record(ctx, totalBytesPerTicket.M(int64(proto.Size(ticket))))

	err := store.CreateTicket(ctx, ticket)
	if err != nil {
		return nil, err
	}

	err = store.IndexTicket(ctx, ticket)
	if err != nil {
		return nil, err
	}

	return ticket, nil
}

// CreateBackfill creates a new Backfill object.
// it assigns an unique Id to the input Backfill and record it in state storage.
// A Backfill is considered as ready for matchmaking once it is created.
//   - If SearchFields exist in a Backfill, CreateBackfill will also index these fields such that one can query the ticket with query.QueryBackfills function.
func (s *frontendService) CreateBackfill(ctx context.Context, req *pb.CreateBackfillRequest) (*pb.Backfill, error) {
	// Perform input validation.
	if req.Backfill == nil {
		return nil, status.Errorf(codes.InvalidArgument, ".backfill is required")
	}
	if req.Backfill.CreateTime != nil {
		return nil, status.Errorf(codes.InvalidArgument, "backfills cannot be created with create time set")
	}

	return doCreateBackfill(ctx, req, s.store)
}

func doCreateBackfill(ctx context.Context, req *pb.CreateBackfillRequest, store statestore.Service) (*pb.Backfill, error) {
	// Generate an id and create a Backfill in state storage
	backfill, ok := proto.Clone(req.Backfill).(*pb.Backfill)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to clone input ticket proto")
	}

	backfill.Id = xid.New().String()
	backfill.CreateTime = ptypes.TimestampNow()


	sfCount := 0
	sfCount += len(backfill.GetSearchFields().GetDoubleArgs())
	sfCount += len(backfill.GetSearchFields().GetStringArgs())
	sfCount += len(backfill.GetSearchFields().GetTags())
	stats.Record(ctx, searchFieldsPerBackfill.M(int64(sfCount)))
	stats.Record(ctx, totalBytesPerBackfill.M(int64(proto.Size(backfill))))

	err := store.CreateBackfill(ctx, backfill, []string{})
	if err != nil {
		return nil, err

	return backfill, nil
}

// DeleteTicket immediately stops Open Match from using the Ticket for matchmaking and removes the Ticket from state storage.
// The client must delete the Ticket when finished matchmaking with it.
//   - If SearchFields exist in a Ticket, DeleteTicket will deindex the fields lazily.
// Users may still be able to assign/get a ticket after calling DeleteTicket on it.
func (s *frontendService) DeleteTicket(ctx context.Context, req *pb.DeleteTicketRequest) (*empty.Empty, error) {
	err := doDeleteTicket(ctx, req.GetTicketId(), s.store)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func doDeleteTicket(ctx context.Context, id string, store statestore.Service) error {
	// Deindex this Ticket to remove it from matchmaking pool.
	err := store.DeindexTicket(ctx, id)
	if err != nil {
		return err
	}

	//'lazy' ticket delete that should be called after a ticket
	// has been deindexed.
	go func() {
		ctx, span := trace.StartSpan(context.Background(), "open-match/frontend.DeleteTicketLazy")
		defer span.End()
		err := store.DeleteTicket(ctx, id)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error": err.Error(),
				"id":    id,
			}).Error("failed to delete the ticket")
		}
		err = store.DeleteTicketsFromPendingRelease(ctx, []string{id})
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error": err.Error(),
				"id":    id,
			}).Error("failed to delete the ticket from pendingRelease")
		}
		// TODO: If other redis queues are implemented or we have custom index fields
		// created by Open Match, those need to be cleaned up here.
	}()
	return nil
}

// GetTicket get the Ticket associated with the specified TicketId.
func (s *frontendService) GetTicket(ctx context.Context, req *pb.GetTicketRequest) (*pb.Ticket, error) {
	return s.store.GetTicket(ctx, req.GetTicketId())
}

// WatchAssignments stream back Assignment of the specified TicketId if it is updated.
//   - If the Assignment is not updated, GetAssignment will retry using the configured backoff strategy.
func (s *frontendService) WatchAssignments(req *pb.WatchAssignmentsRequest, stream pb.FrontendService_WatchAssignmentsServer) error {
	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sender := func(assignment *pb.Assignment) error {
				return stream.Send(&pb.WatchAssignmentsResponse{Assignment: assignment})
			}
			return doWatchAssignments(ctx, req.GetTicketId(), sender, s.store)
		}
	}
}

func doWatchAssignments(ctx context.Context, id string, sender func(*pb.Assignment) error, store statestore.Service) error {
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
				return status.Errorf(codes.Aborted, err.Error())
			}
		}
		return nil
	}

	return store.GetAssignments(ctx, id, callback)
}

// AcknowledgeBackfill is used to notify OpenMatch about GameServer connection info.
// This triggers an assignment process.
func (s *frontendService) AcknowledgeBackfill(ctx context.Context, req *pb.AcknowledgeBackfillRequest) (*pb.Backfill, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// DeleteBackfill deletes a Backfill by its ID.
func (s *frontendService) DeleteBackfill(ctx context.Context, req *pb.DeleteBackfillRequest) (*empty.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// GetBackfill fetches a Backfill object by its ID.
func (s *frontendService) GetBackfill(ctx context.Context, req *pb.GetBackfillRequest) (*pb.Backfill, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// UpdateBackfill updates a Backfill object, if present.
func (s *frontendService) UpdateBackfill(ctx context.Context, req *pb.UpdateBackfillRequest) (*pb.Backfill, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

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

package statestore

import (
	"context"

	"go.opencensus.io/trace"
	"open-match.dev/open-match/pkg/pb"
)

// instrumentedService is a wrapper for a statestore service that provides instrumentation (metrics and tracing) of the database.
type instrumentedService struct {
	s Service
}

func (is *instrumentedService) Close() error {
	return is.s.Close()
}

func (is *instrumentedService) HealthCheck(ctx context.Context) error {
	err := is.s.HealthCheck(ctx)
	return err
}

func (is *instrumentedService) CreateTicket(ctx context.Context, ticket *pb.Ticket) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.CreateTicket")
	defer span.End()
	return is.s.CreateTicket(ctx, ticket)
}

func (is *instrumentedService) GetTicket(ctx context.Context, id string) (*pb.Ticket, error) {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.GetTicket")
	defer span.End()
	return is.s.GetTicket(ctx, id)
}

func (is *instrumentedService) DeleteTicket(ctx context.Context, id string) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.DeleteTicket")
	defer span.End()
	return is.s.DeleteTicket(ctx, id)
}

func (is *instrumentedService) IndexTicket(ctx context.Context, ticket *pb.Ticket) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.IndexTicket")
	defer span.End()
	return is.s.IndexTicket(ctx, ticket)
}

func (is *instrumentedService) DeindexTicket(ctx context.Context, id string) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.DeindexTicket")
	defer span.End()
	return is.s.DeindexTicket(ctx, id)
}

func (is *instrumentedService) GetTickets(ctx context.Context, ids []string) ([]*pb.Ticket, error) {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.GetTickets")
	defer span.End()
	return is.s.GetTickets(ctx, ids)
}

func (is *instrumentedService) GetIndexedIDSet(ctx context.Context) (map[string]struct{}, error) {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.GetIndexedIDSet")
	defer span.End()
	return is.s.GetIndexedIDSet(ctx)
}

func (is *instrumentedService) UpdateAssignments(ctx context.Context, req *pb.AssignTicketsRequest) (*pb.AssignTicketsResponse, []*pb.Ticket, error) {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.UpdateAssignments")
	defer span.End()
	return is.s.UpdateAssignments(ctx, req)
}

func (is *instrumentedService) GetAssignments(ctx context.Context, id string, callback func(*pb.Assignment) error) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.GetAssignments")
	defer span.End()
	return is.s.GetAssignments(ctx, id, callback)
}

func (is *instrumentedService) AddTicketsToPendingRelease(ctx context.Context, ids []string) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.AddTicketsToPendingRelease")
	defer span.End()
	return is.s.AddTicketsToPendingRelease(ctx, ids)
}

func (is *instrumentedService) DeleteTicketsFromPendingRelease(ctx context.Context, ids []string) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.DeleteTicketsFromPendingRelease")
	defer span.End()
	return is.s.DeleteTicketsFromPendingRelease(ctx, ids)
}

func (is *instrumentedService) ReleaseAllTickets(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.ReleaseAllTickets")
	defer span.End()
	return is.s.ReleaseAllTickets(ctx)
}

// CreateBackfill creates a new Backfill in the state storage if one doesn't exist. The xids algorithm used to create the ids ensures that they are unique with no system wide synchronization. Calling clients are forbidden from choosing an id during create. So no conflicts will occur.
func (is *instrumentedService) CreateBackfill(ctx context.Context, backfill *pb.Backfill, ticketIDs []string) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.CreateBackfill")
	defer span.End()
	return is.s.CreateBackfill(ctx, backfill, ticketIDs)
}

// GetBackfill gets the Backfill with the specified id from state storage. This method fails if the Backfill does not exist. Returns the Backfill and associated ticketIDs if they exist.
func (is *instrumentedService) GetBackfill(ctx context.Context, id string) (*pb.Backfill, []string, error) {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.GetBackfill")
	defer span.End()
	return is.s.GetBackfill(ctx, id)
}

// GetBackfills returns multiple backfills from storage.
func (is *instrumentedService) GetBackfills(ctx context.Context, ids []string) ([]*pb.Backfill, error) {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.GetBackfills")
	defer span.End()
	return is.s.GetBackfills(ctx, ids)
}

// DeleteBackfill removes the Backfill with the specified id from state storage. This method succeeds if the Backfill does not exist.
func (is *instrumentedService) DeleteBackfill(ctx context.Context, id string) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.DeleteBackfill")
	defer span.End()
	return is.s.DeleteBackfill(ctx, id)
}

// UpdateBackfill updates an existing Backfill with a new data. ticketIDs can be nil.
func (is *instrumentedService) UpdateBackfill(ctx context.Context, backfill *pb.Backfill, ticketIDs []string) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.UpdateBackfill")
	defer span.End()
	return is.s.UpdateBackfill(ctx, backfill, ticketIDs)
}

// NewMutex returns a new distributed mutex with given name
func (is *instrumentedService) NewMutex(key string) RedisLocker {
	_, span := trace.StartSpan(context.Background(), "statestore/instrumented.NewMutex")
	defer span.End()
	return is.s.NewMutex(key)
}

// UpdateAcknowledgmentTimestamp stores Backfill's last acknowledged time
func (is *instrumentedService) UpdateAcknowledgmentTimestamp(ctx context.Context, id string) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.UpdateAcknowledgmentTimestamp")
	defer span.End()
	return is.s.UpdateAcknowledgmentTimestamp(ctx, id)
}

// GetExpiredBackfillIDs - get all backfills which are expired
func (is *instrumentedService) GetExpiredBackfillIDs(ctx context.Context) ([]string, error) {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.GetExpiredBackfillIDs")
	defer span.End()
	return is.s.GetExpiredBackfillIDs(ctx)
}

// IndexBackfill adds the backfill to the index.
func (is *instrumentedService) IndexBackfill(ctx context.Context, backfill *pb.Backfill) error {
	_, span := trace.StartSpan(ctx, "statestore/instrumented.IndexBackfill")
	defer span.End()
	return is.s.IndexBackfill(ctx, backfill)
}

// DeindexBackfill removes specified Backfill ID from the index. The Backfill continues to exist.
func (is *instrumentedService) DeindexBackfill(ctx context.Context, id string) error {
	_, span := trace.StartSpan(ctx, "statestore/instrumented.DeindexBackfill")
	defer span.End()
	return is.s.DeindexBackfill(ctx, id)
}

// GetIndexedBackfills returns the ids of all backfills currently indexed.
func (is *instrumentedService) GetIndexedBackfills(ctx context.Context) (map[string]int, error) {
	_, span := trace.StartSpan(ctx, "statestore/instrumented.GetIndexedBackfills")
	defer span.End()
	return is.s.GetIndexedBackfills(ctx)
}

// CleanupBackfills removes expired backfills
func (is *instrumentedService) CleanupBackfills(ctx context.Context) error {
	_, span := trace.StartSpan(context.Background(), "statestore/instrumented.CleanupBackfills")
	defer span.End()
	return is.s.CleanupBackfills(ctx)
}

// DeleteBackfillCompletely performs a set of operations to remove backfill and all related entities.
func (is *instrumentedService) DeleteBackfillCompletely(ctx context.Context, id string) error {
	_, span := trace.StartSpan(context.Background(), "statestore/instrumented.DeleteBackfillCompletely")
	defer span.End()
	return is.s.DeleteBackfillCompletely(ctx, id)
}

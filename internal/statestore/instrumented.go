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
	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

var (
	mStateStoreCreateTicketCount               = telemetry.Counter("statestore/createticketcount", "number of tickets created")
	mStateStoreGetTicketCount                  = telemetry.Counter("statestore/getticketcount", "number of tickets retrieved")
	mStateStoreDeleteTicketCount               = telemetry.Counter("statestore/deleteticketcount", "number of tickets deleted")
	mStateStoreIndexTicketCount                = telemetry.Counter("statestore/indexticketcount", "number of tickets indexed")
	mStateStoreDeindexTicketCount              = telemetry.Counter("statestore/deindexticketcount", "number of tickets deindexed")
	mStateStoreGetTicketsCount                 = telemetry.Counter("statestore/getticketscount", "number of bulk ticket retrievals")
	mStateStoreGetIndexedIDSetCount            = telemetry.Counter("statestore/getindexedidsetcount", "number of bulk indexed id retrievals")
	mStateStoreUpdateAssignmentsCount          = telemetry.Counter("statestore/updateassignmentcount", "number of tickets assigned")
	mStateStoreGetAssignmentsCount             = telemetry.Counter("statestore/getassignmentscount", "number of ticket assigned retrieved")
	mStateStoreAddTicketsToIgnoreListCount     = telemetry.Counter("statestore/addticketstoignorelistcount", "number of tickets moved to ignore list")
	mStateStoreDeleteTicketFromIgnoreListCount = telemetry.Counter("statestore/deleteticketfromignorelistcount", "number of tickets removed from ignore list")
)

// instrumentedService is a wrapper for a statestore service that provides instrumentation (metrics and tracing) of the database.
type instrumentedService struct {
	s Service
}

// Close the connection to the database.
func (is *instrumentedService) Close() error {
	return is.s.Close()
}

// HealthCheck indicates if the database is reachable.
func (is *instrumentedService) HealthCheck(ctx context.Context) error {
	err := is.s.HealthCheck(ctx)
	return err
}

// CreateTicket creates a new Ticket in the state storage. If the id already exists, it will be overwritten.
func (is *instrumentedService) CreateTicket(ctx context.Context, ticket *pb.Ticket) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.CreateTicket")
	defer span.End()
	defer telemetry.RecordUnitMeasurement(ctx, mStateStoreCreateTicketCount)
	return is.s.CreateTicket(ctx, ticket)
}

// GetTicket gets the Ticket with the specified id from state storage. This method fails if the Ticket does not exist.
func (is *instrumentedService) GetTicket(ctx context.Context, id string) (*pb.Ticket, error) {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.GetTicket")
	defer span.End()
	defer telemetry.RecordUnitMeasurement(ctx, mStateStoreGetTicketCount)
	return is.s.GetTicket(ctx, id)
}

// DeleteTicket removes the Ticket with the specified id from state storage.
func (is *instrumentedService) DeleteTicket(ctx context.Context, id string) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.DeleteTicket")
	defer span.End()
	defer telemetry.RecordUnitMeasurement(ctx, mStateStoreDeleteTicketCount)
	return is.s.DeleteTicket(ctx, id)
}

// IndexTicket indexes the Ticket id for the configured index fields.
func (is *instrumentedService) IndexTicket(ctx context.Context, ticket *pb.Ticket) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.IndexTicket")
	defer span.End()
	defer telemetry.RecordUnitMeasurement(ctx, mStateStoreIndexTicketCount)
	return is.s.IndexTicket(ctx, ticket)
}

// DeindexTicket removes the indexing for the specified Ticket. Only the indexes are removed but the Ticket continues to exist.
func (is *instrumentedService) DeindexTicket(ctx context.Context, id string) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.DeindexTicket")
	defer span.End()
	defer telemetry.RecordUnitMeasurement(ctx, mStateStoreDeindexTicketCount)
	return is.s.DeindexTicket(ctx, id)
}

// GetTickets returns multiple tickets from storage.  Missing tickets are
// silently ignored.
func (is *instrumentedService) GetTickets(ctx context.Context, ids []string) ([]*pb.Ticket, error) {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.GetTickets")
	defer span.End()
	defer telemetry.RecordUnitMeasurement(ctx, mStateStoreGetTicketsCount)
	return is.s.GetTickets(ctx, ids)
}

// GetIndexedIds returns the ids of all tickets currently indexed.
func (is *instrumentedService) GetIndexedIDSet(ctx context.Context) (map[string]struct{}, error) {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.GetIndexedIDSet")
	defer span.End()
	defer telemetry.RecordUnitMeasurement(ctx, mStateStoreGetIndexedIDSetCount)
	return is.s.GetIndexedIDSet(ctx)
}

// UpdateAssignments update the match assignments for the input ticket ids.
// This function guarantees if any of the input ids does not exists, the state of the storage service won't be altered.
// However, since Redis does not support transaction roll backs (see https://redis.io/topics/transactions), some of the
// assignment fields might be partially updated if this function encounters an error halfway through the execution.
func (is *instrumentedService) UpdateAssignments(ctx context.Context, ids []string, assignment *pb.Assignment) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.UpdateAssignments")
	defer span.End()
	defer telemetry.RecordUnitMeasurement(ctx, mStateStoreUpdateAssignmentsCount)
	return is.s.UpdateAssignments(ctx, ids, assignment)
}

// GetAssignments returns the assignment associated with the input ticket id
func (is *instrumentedService) GetAssignments(ctx context.Context, id string, callback func(*pb.Assignment) error) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.GetAssignments")
	defer span.End()
	return is.s.GetAssignments(ctx, id, func(a *pb.Assignment) error {
		defer telemetry.RecordUnitMeasurement(ctx, mStateStoreGetAssignmentsCount)
		return callback(a)
	})
}

// AddTicketsToIgnoreList appends new proposed tickets to the proposed sorted set with current timestamp
func (is *instrumentedService) AddTicketsToIgnoreList(ctx context.Context, ids []string) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.AddTicketsToIgnoreList")
	defer span.End()
	defer telemetry.RecordNUnitMeasurement(ctx, mStateStoreAddTicketsToIgnoreListCount, int64(len(ids)))
	return is.s.AddTicketsToIgnoreList(ctx, ids)
}

// DeleteTicketsFromIgnoreList deletes tickets from the proposed sorted set
func (is *instrumentedService) DeleteTicketsFromIgnoreList(ctx context.Context, ids []string) error {
	ctx, span := trace.StartSpan(ctx, "statestore/instrumented.DeleteTicketsFromIgnoreList")
	defer span.End()
	defer telemetry.RecordNUnitMeasurement(ctx, mStateStoreDeleteTicketFromIgnoreListCount, int64(len(ids)))
	return is.s.DeleteTicketsFromIgnoreList(ctx, ids)
}

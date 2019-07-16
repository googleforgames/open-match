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

	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

var (
	mStateStoreCreateTicket           = telemetry.Counter("statestore/createticket", "tickets created")
	mStateStoreGetTicket              = telemetry.Counter("statestore/getticket", "tickets retrieve")
	mStateStoreDeleteTicket           = telemetry.Counter("statestore/deleteticket", "tickets deleted")
	mStateStoreIndexTicket            = telemetry.Counter("statestore/indexticket", "tickets indexed")
	mStateStoreDeindexTicket          = telemetry.Counter("statestore/deindexticket", "tickets deindexed")
	mStateStoreFilterTickets          = telemetry.Counter("statestore/filterticket", "tickets that were filtered and returned")
	mStateStoreUpdateAssignments      = telemetry.Counter("statestore/updateassignment", "tickets assigned")
	mStateStoreGetAssignments         = telemetry.Counter("statestore/getassignments", "ticket assigned retrieved")
	mStateStoreAddTicketsToIgnoreList = telemetry.Counter("statestore/addticketstoignorelist", "tickets moved to ignore list")
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
	telemetry.IncrementCounter(ctx, mStateStoreCreateTicket)
	return is.s.CreateTicket(ctx, ticket)
}

// GetTicket gets the Ticket with the specified id from state storage. This method fails if the Ticket does not exist.
func (is *instrumentedService) GetTicket(ctx context.Context, id string) (*pb.Ticket, error) {
	telemetry.IncrementCounter(ctx, mStateStoreGetTicket)
	return is.s.GetTicket(ctx, id)
}

// DeleteTicket removes the Ticket with the specified id from state storage.
func (is *instrumentedService) DeleteTicket(ctx context.Context, id string) error {
	telemetry.IncrementCounter(ctx, mStateStoreDeleteTicket)
	return is.s.DeleteTicket(ctx, id)
}

// IndexTicket indexes the Ticket id for the configured index fields.
func (is *instrumentedService) IndexTicket(ctx context.Context, ticket *pb.Ticket) error {
	telemetry.IncrementCounter(ctx, mStateStoreIndexTicket)
	return is.s.IndexTicket(ctx, ticket)
}

// DeindexTicket removes the indexing for the specified Ticket. Only the indexes are removed but the Ticket continues to exist.
func (is *instrumentedService) DeindexTicket(ctx context.Context, id string) error {
	telemetry.IncrementCounter(ctx, mStateStoreDeindexTicket)
	return is.s.DeindexTicket(ctx, id)
}

// FilterTickets returns the Ticket ids and required attribute key-value pairs for the Tickets meeting the specified filtering criteria.
// map[ticket.Id]map[attributeName][attributeValue]
// {
//  "testplayer1": {"ranking" : 56, "loyalty_level": 4},
//  "testplayer2": {"ranking" : 50, "loyalty_level": 3},
// }
func (is *instrumentedService) FilterTickets(ctx context.Context, filters []*pb.Filter, pageSize int, callback func([]*pb.Ticket) error) error {
	return is.s.FilterTickets(ctx, filters, pageSize, func(t []*pb.Ticket) error {
		telemetry.IncrementCounterN(ctx, mStateStoreFilterTickets, len(t))
		return callback(t)
	})
}

// UpdateAssignments update the match assignments for the input ticket ids.
// This function guarantees if any of the input ids does not exists, the state of the storage service won't be altered.
// However, since Redis does not support transaction roll backs (see https://redis.io/topics/transactions), some of the
// assignment fields might be partially updated if this function encounters an error halfway through the execution.
func (is *instrumentedService) UpdateAssignments(ctx context.Context, ids []string, assignment *pb.Assignment) error {
	telemetry.IncrementCounter(ctx, mStateStoreUpdateAssignments)
	return is.s.UpdateAssignments(ctx, ids, assignment)
}

// GetAssignments returns the assignment associated with the input ticket id
func (is *instrumentedService) GetAssignments(ctx context.Context, id string, callback func(*pb.Assignment) error) error {
	return is.s.GetAssignments(ctx, id, func(a *pb.Assignment) error {
		telemetry.IncrementCounter(ctx, mStateStoreGetAssignments)
		return callback(a)
	})
}

// AddProposedTickets appends new proposed tickets to the proposed sorted set with current timestamp
func (is *instrumentedService) AddTicketsToIgnoreList(ctx context.Context, ids []string) error {
	telemetry.IncrementCounterN(ctx, mStateStoreAddTicketsToIgnoreList, len(ids))
	return is.s.AddTicketsToIgnoreList(ctx, ids)
}

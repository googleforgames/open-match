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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/Bose/minisentinel"
	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/gomodule/redigo/redis"
	"github.com/rs/xid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/telemetry"
	utilTesting "open-match.dev/open-match/internal/util/testing"
	"open-match.dev/open-match/pkg/pb"
)

func TestStatestoreSetup(t *testing.T) {
	cfg, closer := createRedis(t, true, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
}

func TestTicketLifecycle(t *testing.T) {
	cfg, closer := createRedis(t, true, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()

	ctx := utilTesting.NewContext(t)

	// Initialize test data
	id := xid.New().String()
	ticket := &pb.Ticket{
		Id: id,
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"testindex1": 42,
			},
		},
		Assignment: &pb.Assignment{
			Connection: "test-tbd",
		},
	}

	// Validate that GetTicket fails for a Ticket that does not exist.
	_, err := service.GetTicket(ctx, id)
	require.NotNil(t, err)
	require.Equal(t, status.Code(err), codes.NotFound)

	// Validate nonexisting Ticket deletion
	err = service.DeleteTicket(ctx, id)
	require.Nil(t, err)

	// Validate nonexisting Ticket deindexing
	err = service.DeindexTicket(ctx, id)
	require.Nil(t, err)

	// Validate Ticket creation
	err = service.CreateTicket(ctx, ticket)
	require.Nil(t, err)

	// Validate Ticket retrival
	result, err := service.GetTicket(ctx, ticket.Id)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, ticket.Id, result.Id)
	require.Equal(t, ticket.SearchFields.DoubleArgs["testindex1"], result.SearchFields.DoubleArgs["testindex1"])
	require.NotNil(t, result.Assignment)
	require.Equal(t, ticket.Assignment.Connection, result.Assignment.Connection)

	// Validate Ticket deletion
	err = service.DeleteTicket(ctx, id)
	require.Nil(t, err)

	_, err = service.GetTicket(ctx, id)
	require.NotNil(t, err)
}

func TestGetAssignmentBeforeSet(t *testing.T) {
	cfg, closer := createRedis(t, true, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	var assignmentResp *pb.Assignment

	err := service.GetAssignments(ctx, "id", func(assignment *pb.Assignment) error {
		assignmentResp = assignment
		return nil
	})
	// GetAssignment failed because the ticket does not exists
	require.Equal(t, status.Convert(err).Code(), codes.NotFound)
	require.Nil(t, assignmentResp)
}

func TestGetAssignmentNormal(t *testing.T) {
	cfg, closer := createRedis(t, true, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	err := service.CreateTicket(ctx, &pb.Ticket{
		Id:         "1",
		Assignment: &pb.Assignment{Connection: "2"},
	})
	require.Nil(t, err)

	var assignmentResp *pb.Assignment
	ctx, cancel := context.WithCancel(ctx)
	callbackCount := 0
	returnedErr := errors.New("some errors")

	err = service.GetAssignments(ctx, "1", func(assignment *pb.Assignment) error {
		assignmentResp = assignment

		if callbackCount == 5 {
			cancel()
			return returnedErr
		} else if callbackCount > 0 {
			// Test the assignment returned was successfully passed in to the callback function
			require.Equal(t, assignmentResp.Connection, "2")
		}

		callbackCount++
		return nil
	})

	// Test GetAssignments was retried for 5 times and returned with expected error
	require.Equal(t, 5, callbackCount)
	require.Equal(t, returnedErr, err)

	// Pass an expired context, err expected
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	err = service.GetAssignments(ctx, "1", func(assignment *pb.Assignment) error { return nil })
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "GetAssignments, id: 1, failed to connect to redis:")
}

func TestUpdateAssignments(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	err := service.CreateTicket(ctx, &pb.Ticket{
		Id:         "1",
		Assignment: &pb.Assignment{Connection: "2"},
	})
	require.Nil(t, err)

	c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port")))
	require.NoError(t, err)
	_, err = c.Do("SET", "wrong-type-key", "wrong-type-value")
	require.NoError(t, err)

	type expected struct {
		resp               *pb.AssignTicketsResponse
		errCode            codes.Code
		errMessage         string
		assignedTicketsIDs []string
	}

	var testCases = []struct {
		description string
		request     *pb.AssignTicketsRequest
		expected
	}{
		{
			description: "no assignments, empty response is returned",
			request:     &pb.AssignTicketsRequest{},
			expected: expected{
				resp:               &pb.AssignTicketsResponse{},
				errCode:            codes.OK,
				errMessage:         "",
				assignedTicketsIDs: []string{},
			},
		},
		{
			description: "updated assignments, no errors",
			request: &pb.AssignTicketsRequest{
				Assignments: []*pb.AssignmentGroup{
					{
						TicketIds:  []string{"1"},
						Assignment: &pb.Assignment{Connection: "2"},
					},
				},
			},
			expected: expected{
				resp:               &pb.AssignTicketsResponse{},
				errCode:            codes.OK,
				errMessage:         "",
				assignedTicketsIDs: []string{"1"},
			},
		},
		{
			description: "nil assignment, error expected",
			request: &pb.AssignTicketsRequest{
				Assignments: []*pb.AssignmentGroup{
					{
						TicketIds:  []string{"1"},
						Assignment: nil,
					},
				},
			},
			expected: expected{
				resp:               nil,
				errCode:            codes.InvalidArgument,
				errMessage:         "AssignmentGroup.Assignment is required",
				assignedTicketsIDs: []string{},
			},
		},
		{
			description: "ticket is assigned multiple times, error expected",
			request: &pb.AssignTicketsRequest{
				Assignments: []*pb.AssignmentGroup{
					{
						TicketIds:  []string{"1"},
						Assignment: &pb.Assignment{Connection: "2"},
					},
					{
						TicketIds:  []string{"1"},
						Assignment: &pb.Assignment{Connection: "2"},
					},
				},
			},
			expected: expected{
				resp:               nil,
				errCode:            codes.InvalidArgument,
				errMessage:         "Ticket id 1 is assigned multiple times in one assign tickets call",
				assignedTicketsIDs: []string{},
			},
		},
		{
			description: "ticket doesn't exist, no error, response failure expected",
			request: &pb.AssignTicketsRequest{
				Assignments: []*pb.AssignmentGroup{
					{
						TicketIds:  []string{"11111"},
						Assignment: &pb.Assignment{Connection: "2"},
					},
				},
			},
			expected: expected{
				resp: &pb.AssignTicketsResponse{
					Failures: []*pb.AssignmentFailure{{
						TicketId: "11111",
						Cause:    pb.AssignmentFailure_TICKET_NOT_FOUND,
					}},
				},
				errCode:            codes.OK,
				errMessage:         "",
				assignedTicketsIDs: []string{},
			},
		},
		{
			description: "wrong value, error expected",
			request: &pb.AssignTicketsRequest{
				Assignments: []*pb.AssignmentGroup{
					{
						TicketIds:  []string{"wrong-type-key"},
						Assignment: &pb.Assignment{Connection: "2"},
					},
				},
			},
			expected: expected{
				resp:               nil,
				errCode:            codes.Internal,
				errMessage:         "failed to unmarshal ticket from redis wrong-type-key",
				assignedTicketsIDs: []string{},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			resp, ticketsAssignedActual, errActual := service.UpdateAssignments(ctx, tc.request)
			if tc.expected.errCode != codes.OK {
				require.Error(t, errActual)
				require.Equal(t, tc.expected.errCode.String(), status.Convert(errActual).Code().String())
				require.Contains(t, status.Convert(errActual).Message(), tc.expected.errMessage)
			} else {
				require.NoError(t, errActual)
				require.Equal(t, tc.expected.resp, resp)
				require.Equal(t, len(tc.expected.assignedTicketsIDs), len(ticketsAssignedActual))

				for _, ticket := range ticketsAssignedActual {
					found := false
					for _, id := range tc.expected.assignedTicketsIDs {
						if ticket.GetId() == id {
							found = true
							break
						}
					}
					require.Truef(t, found, "assigned ticket ID %s is not found in an expected slice", ticket.GetId())
				}
			}
		})
	}

	// Pass an expired context, err expected
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	_, _, err = service.UpdateAssignments(ctx, &pb.AssignTicketsRequest{
		Assignments: []*pb.AssignmentGroup{
			{
				TicketIds:  []string{"11111"},
				Assignment: &pb.Assignment{Connection: "2"},
			},
		},
	})
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "UpdateAssignments, failed to connect to redis: context canceled")
}

func TestConnect(t *testing.T) {
	testConnect(t, false, "")
	testConnect(t, false, "redispassword")
	testConnect(t, true, "")
	testConnect(t, true, "redispassword")
}

func TestHealthCheck(t *testing.T) {
	cfg, closer := createRedis(t, true, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()

	// OK
	ctx := utilTesting.NewContext(t)
	err := service.HealthCheck(ctx)
	require.NoError(t, err)

	// Error expected
	closer()
	err = service.HealthCheck(ctx)
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
}

func TestCreateTicket(t *testing.T) {
	cfg, closer := createRedis(t, true, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	var testCases = []struct {
		description     string
		ticket          *pb.Ticket
		expectedCode    codes.Code
		expectedMessage string
	}{
		{
			description: "ok",
			ticket: &pb.Ticket{
				Id:         "1",
				Assignment: &pb.Assignment{Connection: "2"},
			},
			expectedCode:    codes.OK,
			expectedMessage: "",
		},
		{
			description:     "nil ticket passed, err expected",
			ticket:          nil,
			expectedCode:    codes.Internal,
			expectedMessage: "failed to marshal the ticket proto, id: : proto: Marshal called with nil",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			err := service.CreateTicket(ctx, tc.ticket)
			if tc.expectedCode == codes.OK {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Equal(t, tc.expectedCode.String(), status.Convert(err).Code().String())
				require.Contains(t, status.Convert(err).Message(), tc.expectedMessage)
			}
		})
	}

	// pass an expired context, err expected
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	err := service.CreateTicket(ctx, &pb.Ticket{
		Id:         "222",
		Assignment: &pb.Assignment{Connection: "2"},
	})
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "CreateTicket, id: 222, failed to connect to redis:")
}

func TestGetTicket(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()

	ctx := utilTesting.NewContext(t)

	err := service.CreateTicket(ctx, &pb.Ticket{
		Id:         "mockTicketID",
		Assignment: &pb.Assignment{Connection: "2"},
	})
	require.NoError(t, err)

	c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port")))
	require.NoError(t, err)
	_, err = c.Do("SET", "wrong-type-key", "wrong-type-value")
	require.NoError(t, err)

	var testCases = []struct {
		description     string
		ticketID        string
		expectedCode    codes.Code
		expectedMessage string
	}{
		{
			description:     "ticket is found",
			ticketID:        "mockTicketID",
			expectedCode:    codes.OK,
			expectedMessage: "",
		},
		{
			description:     "empty id passed, err expected",
			ticketID:        "",
			expectedCode:    codes.NotFound,
			expectedMessage: "Ticket id:  not found",
		},
		{
			description:     "wrong id passed, err expected",
			ticketID:        "123456",
			expectedCode:    codes.NotFound,
			expectedMessage: "Ticket id: 123456 not found",
		},
		{
			description:     "item of a wrong type is requested, err expected",
			ticketID:        "wrong-type-key",
			expectedCode:    codes.Internal,
			expectedMessage: "failed to unmarshal the ticket proto, id: wrong-type-key: proto: can't skip unknown wire type",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			ticketActual, errActual := service.GetTicket(ctx, tc.ticketID)
			if tc.expectedCode == codes.OK {
				require.NoError(t, errActual)
				require.NotNil(t, ticketActual)
			} else {
				require.Error(t, errActual)
				require.Equal(t, tc.expectedCode.String(), status.Convert(errActual).Code().String())
				require.Contains(t, status.Convert(errActual).Message(), tc.expectedMessage)
			}
		})
	}

	// pass an expired context, err expected
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	res, err := service.GetTicket(ctx, "12345")
	require.Error(t, err)
	require.Nil(t, res)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "GetTicket, id: 12345, failed to connect to redis:")
}

func TestDeleteTicket(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()

	ctx := utilTesting.NewContext(t)

	err := service.CreateTicket(ctx, &pb.Ticket{
		Id:         "mockTicketID",
		Assignment: &pb.Assignment{Connection: "2"},
	})
	require.NoError(t, err)

	var testCases = []struct {
		description     string
		ticketID        string
		expectedCode    codes.Code
		expectedMessage string
	}{
		{
			description:     "ticket is found and deleted",
			ticketID:        "mockTicketID",
			expectedCode:    codes.OK,
			expectedMessage: "",
		},
		{
			description:     "empty id passed, no err expected",
			ticketID:        "",
			expectedCode:    codes.OK,
			expectedMessage: "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			errActual := service.DeleteTicket(ctx, tc.ticketID)
			if tc.expectedCode == codes.OK {
				require.NoError(t, errActual)

				if tc.ticketID != "" {
					_, errGetTicket := service.GetTicket(ctx, tc.ticketID)
					require.Error(t, errGetTicket)
					require.Equal(t, codes.NotFound.String(), status.Convert(errGetTicket).Code().String())
				}
			} else {
				require.Error(t, errActual)
				require.Equal(t, tc.expectedCode.String(), status.Convert(errActual).Code().String())
				require.Contains(t, status.Convert(errActual).Message(), tc.expectedMessage)
			}
		})
	}

	// pass an expired context, err expected
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	err = service.DeleteTicket(ctx, "12345")
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "DeleteTicket, id: 12345, failed to connect to redis:")
}

func TestIndexTicket(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()

	ctx := utilTesting.NewContext(t)

	generateTickets(ctx, t, service, 2)

	c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port")))
	require.NoError(t, err)
	idsIndexed, err := redis.Strings(c.Do("SMEMBERS", "allTickets"))
	require.NoError(t, err)
	require.Len(t, idsIndexed, 2)
	require.Equal(t, "mockTicketID-0", idsIndexed[0])
	require.Equal(t, "mockTicketID-1", idsIndexed[1])

	// pass an expired context, err expected
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	err = service.IndexTicket(ctx, &pb.Ticket{
		Id:         "12345",
		Assignment: &pb.Assignment{Connection: "2"},
	})
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "IndexTicket, id: 12345, failed to connect to redis:")
}

func TestDeindexTicket(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()

	ctx := utilTesting.NewContext(t)

	generateTickets(ctx, t, service, 2)

	c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port")))
	require.NoError(t, err)
	idsIndexed, err := redis.Strings(c.Do("SMEMBERS", "allTickets"))
	require.NoError(t, err)
	require.Len(t, idsIndexed, 2)
	require.Equal(t, "mockTicketID-0", idsIndexed[0])
	require.Equal(t, "mockTicketID-1", idsIndexed[1])

	// deindex and check that there is only 1 ticket in the returned slice
	err = service.DeindexTicket(ctx, "mockTicketID-1")
	require.NoError(t, err)
	idsIndexed, err = redis.Strings(c.Do("SMEMBERS", "allTickets"))
	require.NoError(t, err)
	require.Len(t, idsIndexed, 1)
	require.Equal(t, "mockTicketID-0", idsIndexed[0])

	// pass an expired context, err expected
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	err = service.DeindexTicket(ctx, "12345")
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "DeindexTicket, id: 12345, failed to connect to redis:")
}

func TestGetIndexedIDSet(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	tickets, _ := generateTickets(ctx, t, service, 2)

	verifyTickets := func(service Service, tickets []*pb.Ticket) {
		ids, err := service.GetIndexedIDSet(ctx)
		require.Nil(t, err)
		require.Equal(t, len(tickets), len(ids))

		for _, tt := range tickets {
			_, ok := ids[tt.GetId()]
			require.True(t, ok)
		}
	}

	// Verify all tickets are created and returned
	verifyTickets(service, tickets)

	c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port")))
	require.NoError(t, err)
	// Add the first ticket to the pending release and verify changes are reflected in the result
	redis.Strings(c.Do("ZADD", "proposed_ticket_ids", time.Now().UnixNano(), "mockTicketID-0"))

	verifyTickets(service, tickets[1:2])

	// Sleep until the pending release expired and verify we still have all the tickets
	time.Sleep(cfg.GetDuration("pendingReleaseTimeout"))
	verifyTickets(service, tickets)

	// Pass an expired context, err expected
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	_, err = service.GetIndexedIDSet(ctx)
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "GetIndexedIDSet, failed to connect to redis:")
}

func TestGetTickets(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	tickets, ids := generateTickets(ctx, t, service, 2)

	res, err := service.GetTickets(ctx, ids)
	require.NoError(t, err)

	for i, tc := range tickets {
		require.Equal(t, tc.GetId(), res[i].GetId())
	}

	// pass empty ids slice
	empty := []string{}
	res, err = service.GetTickets(ctx, empty)
	require.NoError(t, err)
	require.Nil(t, res)

	// pass an expired context, err expected
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	_, err = service.GetTickets(ctx, ids)
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "GetTickets, failed to connect to redis:")
}

func TestDeleteTicketsFromPendingRelease(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	tickets, ids := generateTickets(ctx, t, service, 2)

	verifyTickets := func(service Service, tickets []*pb.Ticket) {
		ids, err := service.GetIndexedIDSet(ctx)
		require.Nil(t, err)
		require.Equal(t, len(tickets), len(ids))

		for _, tt := range tickets {
			_, ok := ids[tt.GetId()]
			require.True(t, ok)
		}
	}

	// Verify all tickets are created and returned
	verifyTickets(service, tickets)

	c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port")))
	require.NoError(t, err)
	// Add the first ticket to the pending release and verify changes are reflected in the result
	redis.Strings(c.Do("ZADD", "proposed_ticket_ids", time.Now().UnixNano(), ids[0]))

	// Verify 1 ticket is indexed
	verifyTickets(service, tickets[1:2])

	require.NoError(t, service.DeleteTicketsFromPendingRelease(ctx, ids[:1]))

	// Verify that ticket is deleted from indexed set
	verifyTickets(service, tickets)

	// Pass an empty ids slice
	empty := []string{}
	require.NoError(t, service.DeleteTicketsFromPendingRelease(ctx, empty))

	// Pass an expired context, err expected
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	err = service.DeleteTicketsFromPendingRelease(ctx, ids)
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "DeleteTicketsFromPendingRelease, failed to connect to redis:")
}

func TestReleaseAllTickets(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	tickets, ids := generateTickets(ctx, t, service, 2)

	verifyTickets := func(service Service, tickets []*pb.Ticket) {
		ids, err := service.GetIndexedIDSet(ctx)
		require.Nil(t, err)
		require.Equal(t, len(tickets), len(ids))

		for _, tt := range tickets {
			_, ok := ids[tt.GetId()]
			require.True(t, ok)
		}
	}

	// Verify all tickets are created and returned
	verifyTickets(service, tickets)

	c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port")))
	require.NoError(t, err)
	// Add the first ticket to the pending release and verify changes are reflected in the result
	redis.Strings(c.Do("ZADD", "proposed_ticket_ids", time.Now().UnixNano(), ids[0]))

	// Verify 1 ticket is indexed
	verifyTickets(service, tickets[1:2])

	require.NoError(t, service.ReleaseAllTickets(ctx))

	// Verify that ticket is deleted from indexed set
	verifyTickets(service, tickets)

	// Pass an expired context, err expected
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	err = service.ReleaseAllTickets(ctx)
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "ReleaseAllTickets, failed to connect to redis:")
}

func TestAddTicketsToPendingRelease(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	tickets, ids := generateTickets(ctx, t, service, 2)

	verifyTickets := func(service Service, tickets []*pb.Ticket) {
		ids, err := service.GetIndexedIDSet(ctx)
		require.Nil(t, err)
		require.Equal(t, len(tickets), len(ids))

		for _, tt := range tickets {
			_, ok := ids[tt.GetId()]
			require.True(t, ok)
		}
	}

	// Verify all tickets are created and returned
	verifyTickets(service, tickets)

	// Add 1st ticket to pending release state
	require.NoError(t, service.AddTicketsToPendingRelease(ctx, ids[:1]))

	// Verify 1 ticket is indexed
	verifyTickets(service, tickets[1:2])

	// Pass an empty ids slice
	empty := []string{}
	require.NoError(t, service.AddTicketsToPendingRelease(ctx, empty))

	// Pass an expired context, err expected
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	err := service.AddTicketsToPendingRelease(ctx, ids)
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "AddTicketsToPendingRelease, failed to connect to redis:")
}

func testConnect(t *testing.T, withSentinel bool, withPassword string) {
	cfg, closer := createRedis(t, withSentinel, withPassword)
	defer closer()
	store := New(cfg)
	defer store.Close()
	ctx := utilTesting.NewContext(t)

	is, ok := store.(*instrumentedService)
	require.True(t, ok)
	rb, ok := is.s.(*redisBackend)
	require.True(t, ok)

	conn, err := rb.redisPool.GetContext(ctx)
	require.NoError(t, err)
	require.NotNil(t, conn)

	rply, err := redis.String(conn.Do("PING"))
	require.Nil(t, err)
	require.Equal(t, "PONG", rply)
}

func createRedis(t *testing.T, withSentinel bool, withPassword string) (config.View, func()) {
	cfg := viper.New()
	closerFuncs := []func(){}
	mredis := miniredis.NewMiniRedis()
	err := mredis.StartAddr("localhost:0")
	if err != nil {
		t.Fatalf("failed to start miniredis, %v", err)
	}
	closerFuncs = append(closerFuncs, mredis.Close)

	cfg.Set("redis.pool.maxIdle", 5)
	cfg.Set("redis.pool.idleTimeout", time.Second)
	cfg.Set("redis.pool.healthCheckTimeout", 100*time.Millisecond)
	cfg.Set("redis.pool.maxActive", 5)
	cfg.Set("pendingReleaseTimeout", "200ms")
	cfg.Set("backoff.initialInterval", 100*time.Millisecond)
	cfg.Set("backoff.randFactor", 0.5)
	cfg.Set("backoff.multiplier", 0.5)
	cfg.Set("backoff.maxInterval", 300*time.Millisecond)
	cfg.Set("backoff.maxElapsedTime", 100*time.Millisecond)
	cfg.Set(telemetry.ConfigNameEnableMetrics, true)
	cfg.Set("assignedDeleteTimeout", 1000*time.Millisecond)

	if withSentinel {
		s := minisentinel.NewSentinel(mredis)
		err = s.StartAddr("localhost:0")
		if err != nil {
			t.Fatalf("failed to start minisentinel, %v", err)
		}

		closerFuncs = append(closerFuncs, s.Close)
		cfg.Set("redis.sentinelHostname", s.Host())
		cfg.Set("redis.sentinelPort", s.Port())
		cfg.Set("redis.sentinelMaster", s.MasterInfo().Name)
		cfg.Set("redis.sentinelEnabled", true)
		// TODO: enable sentinel auth test cases when the library support it.
		cfg.Set("redis.sentinelUsePassword", false)
	} else {
		cfg.Set("redis.hostname", mredis.Host())
		cfg.Set("redis.port", mredis.Port())
	}

	if len(withPassword) > 0 {
		mredis.RequireAuth(withPassword)
		tmpFile, err := ioutil.TempFile("", "password")
		if err != nil {
			t.Fatal("failed to create temp file for password")
		}
		if _, err := tmpFile.WriteString(withPassword); err != nil {
			t.Fatal("failed to write pw to temp file")
		}

		closerFuncs = append(closerFuncs, func() { os.Remove(tmpFile.Name()) })
		cfg.Set("redis.usePassword", true)
		cfg.Set("redis.passwordPath", tmpFile.Name())
	}

	return cfg, func() {
		for _, closer := range closerFuncs {
			closer()
		}
	}
}

//nolint: unparam
// generateTickets creates a proper amount of ticket, returns a slice of tickets and a slice of tickets ids
func generateTickets(ctx context.Context, t *testing.T, service Service, amount int) ([]*pb.Ticket, []string) {
	tickets := make([]*pb.Ticket, 0, amount)
	ids := make([]string, 0, amount)

	for i := 0; i < amount; i++ {
		tmp := &pb.Ticket{
			Id:         fmt.Sprintf("mockTicketID-%d", i),
			Assignment: &pb.Assignment{Connection: "2"},
		}
		require.NoError(t, service.CreateTicket(ctx, tmp))
		require.NoError(t, service.IndexTicket(ctx, tmp))
		tickets = append(tickets, tmp)
		ids = append(ids, tmp.GetId())
	}
	return tickets, ids
}

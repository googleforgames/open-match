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
	"errors"
	"regexp"
	"testing"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	"open-match.dev/open-match/internal/statestore"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
)

func TestDoCreateTickets(t *testing.T) {
	store, closer := statestoreTesting.NewStoreServiceForTesting(t, viper.New())
	defer closer()

	normalCtx := context.Background()
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	testTicket := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"test-property": {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
			},
		},
	}

	tests := []struct {
		description string
		ctx         context.Context
		shouldErr   bool
	}{
		{
			description: "expect error with canceled context",
			ctx:         cancelledCtx,
			shouldErr:   true,
		},
		{
			description: "expect normal return with default context",
			ctx:         normalCtx,
			shouldErr:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			res, err := doCreateTicket(test.ctx, &pb.CreateTicketRequest{Ticket: testTicket}, store)

			assert.Equal(t, test.shouldErr, err != nil)
			if err == nil {
				matched, err := regexp.MatchString(`[0-9a-v]{20}`, res.GetTicket().GetId())
				assert.True(t, matched)
				assert.Nil(t, err)
				assert.Equal(t, testTicket.GetProperties().GetFields()["test-property"].GetNumberValue(), res.GetTicket().GetProperties().GetFields()["test-property"].GetNumberValue())
			}
		})
	}
}

func TestDoGetAssignments(t *testing.T) {
	testTicket := &pb.Ticket{
		Id: "test-id",
	}

	tests := []struct {
		description     string
		preAction       func(*testing.T, statestore.Service, []*pb.Assignment)
		senderGenerator func(chan *pb.Assignment, *int) func(*pb.Assignment) error
		wantCode        codes.Code
		wantAssignments []*pb.Assignment
	}{
		{
			description: "expect error because ticket id does not exist",
			preAction:   func(_ *testing.T, _ statestore.Service, _ []*pb.Assignment) {},
			senderGenerator: func(tmp chan *pb.Assignment, p *int) func(*pb.Assignment) error {
				return func(assignment *pb.Assignment) error {
					*p++
					tmp <- assignment
					return nil
				}
			},
			wantCode:        codes.NotFound,
			wantAssignments: []*pb.Assignment{},
		},
		{
			description: "expect two assignment reads from preAction writes and fail in grpc aborted code",
			preAction: func(t *testing.T, store statestore.Service, wantAssignments []*pb.Assignment) {
				assert.Nil(t, store.CreateTicket(context.Background(), testTicket))
				go func() {
					for i := 0; i < len(wantAssignments); i++ {
						time.Sleep(200 * time.Millisecond)
						assert.Nil(t, store.UpdateAssignments(context.Background(), []string{testTicket.GetId()}, wantAssignments[i]))
					}
				}()
			},
			senderGenerator: func(tmp chan *pb.Assignment, p *int) func(*pb.Assignment) error {
				return func(assignment *pb.Assignment) error {
					*p++
					tmp <- assignment
					if len(tmp) == cap(tmp) {
						return errors.New("some error")
					}
					return nil
				}
			},
			wantCode:        codes.Aborted,
			wantAssignments: []*pb.Assignment{&pb.Assignment{Connection: "1"}, &pb.Assignment{Connection: "2"}},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			store, closer := statestoreTesting.NewStoreServiceForTesting(t, viper.New())
			defer closer()

			senderTriggerCount := 0
			gotAssignmentChan := make(chan *pb.Assignment, len(test.wantAssignments))

			test.preAction(t, store, test.wantAssignments)
			err := doGetAssignments(context.Background(), testTicket.GetId(), test.senderGenerator(gotAssignmentChan, &senderTriggerCount), store)
			assert.Equal(t, test.wantCode, status.Convert(err).Code())

			if err == nil {
				for i := 0; i < cap(gotAssignmentChan); i++ {
					assert.Equal(t, <-gotAssignmentChan, test.wantAssignments[i])
				}
			}
		})
	}
}

// validateTicket validates that the fetched ticket is identical to the expected ticket.
func validateTicket(t *testing.T, got *pb.Ticket, want *pb.Ticket) {
	assert.Equal(t, got.Id, want.Id)
	assert.Equal(t, got.Properties.Fields["test-property"].GetNumberValue(), want.Properties.Fields["test-property"].GetNumberValue())
	assert.Equal(t, got.Assignment.Connection, want.Assignment.Connection)
	assert.Equal(t, got.Assignment.Properties, want.Assignment.Properties)
	assert.Equal(t, got.Assignment.Error, want.Assignment.Error)
}

// validateDelete validates that the ticket is actually deleted from the state storage.
// Given that delete is async, this method retries fetch every 100ms up to 5 seconds.
func validateDelete(t *testing.T, fe pb.FrontendClient, id string) {
	start := time.Now()
	for {
		if time.Since(start) > 5*time.Second {
			break
		}

		// Attempt to fetch the ticket every 100ms
		_, err := fe.GetTicket(context.Background(), &pb.GetTicketRequest{TicketId: id})
		if err != nil {
			// Only failure to fetch with NotFound should be considered as success.
			assert.Equal(t, status.Code(err), codes.NotFound)
			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	assert.Failf(t, "ticket %v not deleted after 5 seconds", id)
}

// TestFrontendService tests creating, getting and deleting a ticket using Frontend service.
func TestFrontendService(t *testing.T) {
	assert := assert.New(t)

	tc := createStore(t)
	fe := pb.NewFrontendClient(tc.MustGRPC())
	assert.NotNil(fe)

	ticket := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"test-property": {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
			},
		},
		Assignment: &pb.Assignment{
			Connection: "test-tbd",
		},
	}

	// Create a ticket, validate that it got an id and set its id in the expected ticket.
	resp, err := fe.CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: ticket})
	assert.NotNil(resp)
	assert.Nil(err)
	want := resp.Ticket
	assert.NotNil(want)
	assert.NotNil(want.Id)
	ticket.Id = want.Id
	validateTicket(t, resp.Ticket, ticket)

	// Fetch the ticket and validate that it is identical to the expected ticket.
	gotTicket, err := fe.GetTicket(context.Background(), &pb.GetTicketRequest{TicketId: ticket.Id})
	assert.NotNil(gotTicket)
	assert.Nil(err)
	validateTicket(t, gotTicket, ticket)

	// Delete the ticket and validate that it was actually deleted.
	_, err = fe.DeleteTicket(context.Background(), &pb.DeleteTicketRequest{TicketId: ticket.Id})
	assert.Nil(err)
	validateDelete(t, fe, ticket.Id)
}

func createStore(t *testing.T) *rpcTesting.TestContext {
	var closer func()
	tc := rpcTesting.MustServe(t, func(p *rpc.ServerParams) {
		cfg := viper.New()
		cfg.Set("playerIndices", []string{"testindex1", "testindex2"})
		closer = statestoreTesting.New(t, cfg)
		err := BindService(p, cfg)
		assert.Nil(t, err)
	})
	tc.AddCloseFunc(closer)
	return tc
}

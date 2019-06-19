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
	"open-match.dev/open-match/internal/statestore"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
	"open-match.dev/open-match/pkg/pb"
)

func TestDoCreateTickets(t *testing.T) {
	cfg := viper.New()
	cfg.Set("playerIndices", []string{"test-property"})
	store, closer := statestoreTesting.NewStoreServiceForTesting(t, cfg)
	defer closer()

	normalCtx := context.Background()
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	numTicket := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"test-property": {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
			},
		},
	}

	stringTicket := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"test-property": {Kind: &structpb.Value_StringValue{StringValue: "string"}},
			},
		},
	}

	tests := []struct {
		description string
		ticket      *pb.Ticket
		ctx         context.Context
		wantErr     bool
	}{
		{
			description: "expect precondition error since open-match does not support properties other than number",
			ctx:         normalCtx,
			ticket:      stringTicket,
			wantErr:     true,
		},
		{
			description: "expect error with canceled context",
			ctx:         cancelledCtx,
			ticket:      numTicket,
			wantErr:     true,
		},
		{
			description: "expect normal return with default context",
			ctx:         normalCtx,
			ticket:      numTicket,
			wantErr:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			res, err := doCreateTicket(test.ctx, &pb.CreateTicketRequest{Ticket: test.ticket}, store)

			assert.Equal(t, test.wantErr, err != nil)
			if err == nil {
				matched, err := regexp.MatchString(`[0-9a-v]{20}`, res.GetTicket().GetId())
				assert.True(t, matched)
				assert.Nil(t, err)
				assert.Equal(t, test.ticket.GetProperties().GetFields()["test-property"].GetNumberValue(), res.GetTicket().GetProperties().GetFields()["test-property"].GetNumberValue())
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
			wantAssignments: []*pb.Assignment{{Connection: "1"}, {Connection: "2"}},
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

func TestDoDeleteTicket(t *testing.T) {
	fakeTicket := &pb.Ticket{
		Id: "1",
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"test-property": {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
			},
		},
	}

	tests := []struct {
		description string
		preAction   func(context.Context, context.CancelFunc, statestore.Service)
		wantCode    codes.Code
	}{
		{
			description: "expect unavailable code since context is canceled before being called",
			preAction: func(_ context.Context, cancel context.CancelFunc, _ statestore.Service) {
				cancel()
			},
			wantCode: codes.Unavailable,
		},
		{
			description: "expect ok code since delete ticket does not care about if ticket exists or not",
			preAction:   func(_ context.Context, _ context.CancelFunc, _ statestore.Service) {},
			wantCode:    codes.OK,
		},
		{
			description: "expect ok code",
			preAction: func(ctx context.Context, _ context.CancelFunc, store statestore.Service) {
				store.CreateTicket(ctx, fakeTicket)
				store.IndexTicket(ctx, fakeTicket)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			store, closer := statestoreTesting.NewStoreServiceForTesting(t, viper.New())
			defer closer()

			test.preAction(ctx, cancel, store)

			err := doDeleteTicket(ctx, fakeTicket.GetId(), store)
			assert.Equal(t, test.wantCode, status.Convert(err).Code())
		})
	}
}

func TestDoGetTicket(t *testing.T) {
	fakeTicket := &pb.Ticket{
		Id: "1",
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"test-property": {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
			},
		},
	}

	tests := []struct {
		description string
		preAction   func(context.Context, context.CancelFunc, statestore.Service)
		wantTicket  *pb.Ticket
		wantCode    codes.Code
	}{
		{
			description: "expect unavailable code since context is canceled before being called",
			preAction: func(_ context.Context, cancel context.CancelFunc, _ statestore.Service) {
				cancel()
			},
			wantCode: codes.Unavailable,
		},
		{
			description: "expect not found code since ticket does not exist",
			preAction:   func(_ context.Context, _ context.CancelFunc, _ statestore.Service) {},
			wantCode:    codes.NotFound,
		},
		{
			description: "expect ok code with output ticket equivalent to fakeTicket",
			preAction: func(ctx context.Context, _ context.CancelFunc, store statestore.Service) {
				store.CreateTicket(ctx, fakeTicket)
				store.IndexTicket(ctx, fakeTicket)
			},
			wantCode:   codes.OK,
			wantTicket: fakeTicket,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			store, closer := statestoreTesting.NewStoreServiceForTesting(t, viper.New())
			defer closer()

			test.preAction(ctx, cancel, store)

			ticket, err := doGetTickets(ctx, fakeTicket.GetId(), store)
			assert.Equal(t, test.wantCode, status.Convert(err).Code())

			if err == nil {
				assert.Equal(t, test.wantTicket.GetId(), ticket.GetId())

				for wantK, wantV := range test.wantTicket.GetProperties().GetFields() {
					actualV, ok := ticket.GetProperties().GetFields()[wantK]
					assert.True(t, ok)
					assert.Equal(t, wantV.GetKind(), actualV.GetKind())
				}
			}
		})
	}
}

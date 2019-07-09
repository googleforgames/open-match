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

package mmlogic

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/statestore"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
	internalTesting "open-match.dev/open-match/internal/testing"
	"open-match.dev/open-match/pkg/pb"
)

func TestDoQueryTickets(t *testing.T) {
	const (
		attribute1 = "level"
		attribute2 = "spd"
	)

	var actualTickets []*pb.Ticket
	fakeErr := errors.New("some error")

	senderGenerator := func(err error) func(tickets []*pb.Ticket) error {
		return func(tickets []*pb.Ticket) error {
			if err != nil {
				return err
			}
			actualTickets = tickets
			return err
		}
	}

	testTickets := internalTesting.GenerateTickets(
		internalTesting.Property{Name: attribute1, Min: 0, Max: 20, Interval: 5},
		internalTesting.Property{Name: attribute2, Min: 0, Max: 20, Interval: 5},
	)

	tests := []struct {
		description string
		sender      func(tickets []*pb.Ticket) error
		filters     []*pb.Filter
		pageSize    int
		action      func(*testing.T, statestore.Service)
		wantErr     error
		wantTickets []*pb.Ticket
	}{
		{
			"expect empty response from an empty store",
			senderGenerator(nil),
			[]*pb.Filter{
				{
					Attribute: attribute1,
					Min:       0,
					Max:       10,
				},
			},
			100,
			func(_ *testing.T, _ statestore.Service) {},
			nil,
			nil,
		},
		{
			"expect tickets with attribute1 value in range of [0, 10] (inclusively)",
			senderGenerator(nil),
			[]*pb.Filter{
				{
					Attribute: attribute1,
					Min:       0,
					Max:       10,
				},
			},
			100,
			func(t *testing.T, store statestore.Service) {
				for _, testTicket := range testTickets {
					assert.Nil(t, store.CreateTicket(context.Background(), testTicket))
					assert.Nil(t, store.IndexTicket(context.Background(), testTicket))
				}
			},
			nil,
			internalTesting.GenerateTickets(
				internalTesting.Property{Name: attribute1, Min: 0, Max: 10.1, Interval: 5},
				internalTesting.Property{Name: attribute2, Min: 0, Max: 20, Interval: 5},
			),
		},
		{
			"expect error from canceled context",
			senderGenerator(fakeErr),
			[]*pb.Filter{
				{
					Attribute: attribute1,
					Min:       0,
					Max:       10,
				},
			},
			100,
			func(t *testing.T, store statestore.Service) {
				for _, testTicket := range testTickets {
					assert.Nil(t, store.CreateTicket(context.Background(), testTicket))
					assert.Nil(t, store.IndexTicket(context.Background(), testTicket))
				}
			},
			status.Errorf(codes.Internal, "%v", fakeErr),
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cfg := viper.New()
			cfg.Set("storage.page.size", 1000)
			cfg.Set("ticketIndices", []string{attribute1, attribute2})
			store, closer := statestoreTesting.NewStoreServiceForTesting(t, cfg)
			defer closer()

			test.action(t, store)
			assert.Equal(t, test.wantErr, doQueryTickets(context.Background(), test.filters, test.pageSize, test.sender, store))
			for _, wantTicket := range test.wantTickets {
				assert.Contains(t, actualTickets, wantTicket)
			}
		})
	}
}

func TestGetPageSize(t *testing.T) {
	testCases := []struct {
		name      string
		configure func(config.Mutable)
		expected  int
	}{
		{
			"notSet",
			func(cfg config.Mutable) {},
			1000,
		},
		{
			"set",
			func(cfg config.Mutable) {
				cfg.Set("storage.page.size", "2156")
			},
			2156,
		},
		{
			"low",
			func(cfg config.Mutable) {
				cfg.Set("storage.page.size", "9")
			},
			10,
		},
		{
			"high",
			func(cfg config.Mutable) {
				cfg.Set("storage.page.size", "10001")
			},
			10000,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := viper.New()
			tt.configure(cfg)
			actual := getPageSize(cfg)
			if actual != tt.expected {
				t.Errorf("got %d, want %d", actual, tt.expected)
			}
		})
	}
}

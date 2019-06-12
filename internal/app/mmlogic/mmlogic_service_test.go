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
	"fmt"
	"io"
	"testing"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
)

type propertyManifest struct {
	name     string
	min      float64
	max      float64
	interval float64
}

func TestDoQueryTickets(t *testing.T) {
	assert := assert.New(t)

	const (
		attribute1 = "level"
		attribute2 = "spd"
	)

	ctx := context.Background()
	cfg := viper.New()
	cfg.Set("storage.page.size", 1000)
	cfg.Set("playerIndices", []string{attribute1, attribute2})
	store, closer := statestoreTesting.NewStoreServiceForTesting(t, cfg)
	defer closer()

	var actualTickets []*pb.Ticket

	sender := func(tickets []*pb.Ticket) error {
		actualTickets = tickets
		return nil
	}

	testTickets := generateTickets(propertyManifest{attribute1, 0, 20, 5}, propertyManifest{attribute2, 0, 20, 5})

	tests := []struct {
		filters       []*pb.Filter
		pageSize      int
		action        func() error
		shouldErr     error
		shouldTickets []*pb.Ticket
	}{
		{
			[]*pb.Filter{
				{
					Attribute: attribute1,
					Min:       0,
					Max:       10,
				},
			},
			100,
			func() error { return nil },
			nil,
			nil,
		},
		{
			[]*pb.Filter{
				{
					Attribute: attribute1,
					Min:       0,
					Max:       10,
				},
			},
			100,
			func() error {
				for _, testTicket := range testTickets {
					assert.Nil(store.CreateTicket(ctx, testTicket))
					assert.Nil(store.IndexTicket(ctx, testTicket))
				}
				return nil
			},
			nil,
			generateTickets(propertyManifest{attribute1, 0, 10.1, 5}, propertyManifest{attribute2, 0, 20, 5}),
		},
	}

	for _, test := range tests {
		assert.Nil(test.action())
		assert.Equal(test.shouldErr, doQueryTickets(ctx, test.filters, test.pageSize, sender, store))
		for _, shouldTicket := range test.shouldTickets {
			assert.Contains(actualTickets, shouldTicket)
		}
	}
}

func generateTickets(manifest1, manifest2 propertyManifest) []*pb.Ticket {
	testTickets := make([]*pb.Ticket, 0)

	for i := manifest1.min; i < manifest1.max; i += manifest1.interval {
		for j := manifest2.min; j < manifest2.max; j += manifest2.interval {
			testTickets = append(testTickets, &pb.Ticket{
				Id: fmt.Sprintf("%s%f-%s%f", manifest1.name, i, manifest2.name, j),
				Properties: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						manifest1.name: {Kind: &structpb.Value_NumberValue{NumberValue: i}},
						manifest2.name: {Kind: &structpb.Value_NumberValue{NumberValue: j}},
					},
				},
			})
		}
	}

	return testTickets
}

// TODO: move below test cases to e2e
func TestQueryTicketsEmptyRequest(t *testing.T) {
	assert := assert.New(t)
	tc := createMmlogicForTest(t)
	defer tc.Close()

	queryTicketsLoop(t, tc, &pb.QueryTicketsRequest{}, func(_ *pb.QueryTicketsResponse, err error) {
		assert.Equal(codes.InvalidArgument, status.Convert(err).Code())
	})
}

func TestQueryTicketsForEmptyDatabase(t *testing.T) {
	assert := assert.New(t)
	tc := createMmlogicForTest(t)
	defer tc.Close()

	queryTicketsLoop(t, tc,
		&pb.QueryTicketsRequest{
			Pool: &pb.Pool{
				Filter: []*pb.Filter{{
					Attribute: "ok",
				}},
			},
		},
		func(resp *pb.QueryTicketsResponse, err error) {
			assert.NotNil(resp)
		})
}

func queryTicketsLoop(t *testing.T, tc *rpcTesting.TestContext, req *pb.QueryTicketsRequest, handleResponse func(*pb.QueryTicketsResponse, error)) {
	c := pb.NewMmLogicClient(tc.MustGRPC())
	stream, err := c.QueryTickets(tc.Context(), req)
	if err != nil {
		t.Fatalf("error querying tickets, %v", err)
	}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		handleResponse(resp, err)
		if err != nil {
			return
		}
	}
}

func createMmlogicForTest(t *testing.T) *rpcTesting.TestContext {
	var closerFunc func()
	tc := rpcTesting.MustServe(t, func(p *rpc.ServerParams) {
		cfg := viper.New()
		closerFunc = statestoreTesting.New(t, cfg)
		cfg.Set("storage.page.size", 10)

		BindService(p, cfg)
	})
	// TODO: This is very ugly. Need a better story around closing resources.
	tc.AddCloseFunc(closerFunc)
	return tc
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

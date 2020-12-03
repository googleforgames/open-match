// Copyright 2020 Google LLC
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

package e2e

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/pkg/pb"
)

// TestCreateGetBackfill Create and Get Backfill test
func TestCreateGetBackfill(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	bf := &pb.CreateBackfillRequest{Backfill: &pb.Backfill{SearchFields: &pb.SearchFields{
		StringArgs: map[string]string{
			"search": "me",
		},
	}}}
	b1, err := om.Frontend().CreateBackfill(ctx, bf)
	require.Nil(t, err)
	require.NotNil(t, b1)

	b2, err := om.Frontend().CreateBackfill(ctx, bf)
	require.Nil(t, err)
	require.NotNil(t, b2)

	// Different IDs should be generated
	require.NotEqual(t, b1.Id, b2.Id)
	matched, err := regexp.MatchString(`[0-9a-v]{20}`, b1.GetId())
	require.True(t, matched)
	require.Nil(t, err)
	require.Equal(t, bf.Backfill.SearchFields.DoubleArgs["test-arg"], b1.SearchFields.DoubleArgs["test-arg"])
	b1.Id = b2.Id
	b1.CreateTime = b2.CreateTime

	// All fields other than CreateTime and Id fields should be equal
	require.Equal(t, b1, b2)
	require.Nil(t, err)
	actual, err := om.Frontend().GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: b1.Id})
	require.Nil(t, err)
	require.NotNil(t, actual)
	require.Equal(t, b1, actual)

	// Get Backfill which is not present - NotFound
	actual, err = om.Frontend().GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: b1.Id + "new"})
	require.NotNil(t, err)
	require.Equal(t, codes.NotFound.String(), status.Convert(err).Code().String())
	require.Nil(t, actual)
}

func TestProposedBackfillCreate(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{
		Ticket: &pb.Ticket{
			SearchFields: &pb.SearchFields{
				StringArgs: map[string]string{
					"field": "value",
				},
			},
		},
	})
	require.Nil(t, err)

	b := &pb.Backfill{
		SearchFields: &pb.SearchFields{
			StringArgs: map[string]string{
				"field": "value",
			},
		},
	}
	m := &pb.Match{
		MatchId:  "1",
		Tickets:  []*pb.Ticket{t1},
		Backfill: b,
	}

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		out <- m
		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		p, ok := <-in
		require.True(t, ok)

		out <- p.MatchId
		return nil
	})

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	resp, err := stream.Recv()
	require.Nil(t, err)
	bfID := resp.Match.Backfill.Id

	resp, err = stream.Recv()
	require.Nil(t, resp)
	require.Equal(t, io.EOF, err)

	actual, err := om.Frontend().GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: bfID})
	require.Nil(t, err)
	require.NotNil(t, actual)

	b.Id = actual.Id
	b.CreateTime = actual.CreateTime
	require.True(t, proto.Equal(b, actual))

	client, err := om.Query().QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{
		StringEqualsFilters: []*pb.StringEqualsFilter{{StringArg: "field", Value: "value"}},
	}})

	require.Nil(t, err)
	_, err = client.Recv()
	require.Equal(t, io.EOF, err)
}

func TestProposedBackfillUpdate(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)
	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{
		Ticket: &pb.Ticket{
			SearchFields: &pb.SearchFields{
				StringArgs: map[string]string{
					"field": "value",
				},
			},
		},
	})
	require.Nil(t, err)

	b, err := om.Frontend().CreateBackfill(ctx, &pb.CreateBackfillRequest{Backfill: &pb.Backfill{
		SearchFields: &pb.SearchFields{
			StringArgs: map[string]string{
				"field": "value",
			},
		},
	}})
	require.Nil(t, err)

	b.SearchFields = &pb.SearchFields{
		StringArgs: map[string]string{
			"field1": "value1",
		},
	}
	//using DefaultEvaluationCriteria just for testing purposes only
	b.Extensions = map[string]*any.Any{
		"evaluation_input": mustAny(&pb.DefaultEvaluationCriteria{
			Score: 10,
		}),
	}
	m := &pb.Match{
		MatchId:  "1",
		Tickets:  []*pb.Ticket{t1},
		Backfill: b,
	}

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		out <- m
		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		p, ok := <-in
		require.True(t, ok)

		out <- p.MatchId
		return nil
	})

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	resp, err := stream.Recv()
	require.Nil(t, err)
	require.True(t, proto.Equal(m, resp.Match))

	resp, err = stream.Recv()
	require.Nil(t, resp)
	require.Equal(t, io.EOF, err)

	actual, err := om.Frontend().GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: b.Id})
	require.Nil(t, err)
	require.NotNil(t, actual)
	require.True(t, proto.Equal(b, actual))

	client, err := om.Query().QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{
		StringEqualsFilters: []*pb.StringEqualsFilter{{StringArg: "field", Value: "value"}},
	}})
	require.Nil(t, err)
	_, err = client.Recv()
	require.Equal(t, io.EOF, err)
}

func TestBackfillGenerationMismatch(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)
	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	b, err := om.Frontend().CreateBackfill(ctx, &pb.CreateBackfillRequest{Backfill: &pb.Backfill{Generation: 1}})
	require.Nil(t, err)

	b.Generation = 0
	m := &pb.Match{
		MatchId:  "1",
		Tickets:  []*pb.Ticket{t1},
		Backfill: b,
	}

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		out <- m
		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		p, ok := <-in
		require.True(t, ok)

		out <- p.MatchId
		return nil
	})

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	resp, err := stream.Recv()
	require.Nil(t, resp)
	require.Equal(t, io.EOF, err)
}

func TestCleanUpBackfills(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	om := newOM(t)

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)
	require.NotNil(t, t1)

	b1, err := om.Frontend().CreateBackfill(ctx, &pb.CreateBackfillRequest{Backfill: &pb.Backfill{
		SearchFields: &pb.SearchFields{
			StringArgs: map[string]string{
				"search": "me",
			},
		}}})
	require.Nil(t, err)
	require.NotNil(t, b1)
	fmt.Println(b1.Id)

	m := &pb.Match{
		MatchId:  "1",
		Tickets:  []*pb.Ticket{t1},
		Backfill: b1,
	}

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		out <- m
		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		p, ok := <-in
		require.True(t, ok)
		require.True(t, proto.Equal(p, m))
		_, ok = <-in
		require.False(t, ok)

		out <- m.MatchId
		return nil
	})

	// wait until backfill is expired, then try to get it
	time.Sleep(3 * time.Second)

	// statestore.CleanupBackfills is called at the end of each syncronizer cycle after fetch matches call, so expired backfill will be removed
	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	_, err = stream.Recv()
	require.Error(t, err)
	e, ok := status.FromError(err)
	require.True(t, ok)
	require.Contains(t, e.Message(), "error(s) in FetchMatches call. syncErr=[failed to handle match backfill: 1: rpc error: code = NotFound desc = Backfill id:")
}

func mustAny(m proto.Message) *any.Any {
	result, err := ptypes.MarshalAny(m)
	if err != nil {
		panic(err)
	}
	return result
}

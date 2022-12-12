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
	"google.golang.org/protobuf/types/known/anypb"
	"open-match.dev/open-match/pkg/pb"
)

func TestCreateGetBackfill(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	bf := &pb.CreateBackfillRequest{Backfill: &pb.Backfill{SearchFields: &pb.SearchFields{
		StringArgs: map[string]string{
			"search": "me",
		},
	}}}
	b1, err := om.Frontend().CreateBackfill(ctx, bf)
	require.NoError(t, err)
	require.NotNil(t, b1)

	b2, err := om.Frontend().CreateBackfill(ctx, bf)
	require.NoError(t, err)
	require.NotNil(t, b2)

	// Different IDs should be generated
	require.NotEqual(t, b1.Id, b2.Id)
	matched, err := regexp.MatchString(`[0-9a-v]{20}`, b1.GetId())
	require.True(t, matched)
	require.NoError(t, err)
	require.Equal(t, bf.Backfill.SearchFields.DoubleArgs["test-arg"], b1.SearchFields.DoubleArgs["test-arg"])
	b1.Id = b2.Id
	b1.CreateTime = b2.CreateTime

	// All fields other than CreateTime and Id fields should be equal
	require.Equal(t, b1, b2)
	require.NoError(t, err)
	actual, err := om.Frontend().GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: b1.Id})
	require.NoError(t, err)
	require.NotNil(t, actual)
	require.Equal(t, b1, actual)

	// Get Backfill which is not present - NotFound
	actual, err = om.Frontend().GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: b1.Id + "new"})
	require.NotNil(t, err)
	require.Equal(t, codes.NotFound.String(), status.Convert(err).Code().String())
	require.Nil(t, actual)
}

// TestBackfillFrontendLifecycle Create, Get and Update Backfill test
func TestBackfillFrontendLifecycle(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	bf := &pb.Backfill{SearchFields: &pb.SearchFields{
		StringArgs: map[string]string{
			"search": "me",
		},
	}}

	createdBf, err := om.Frontend().CreateBackfill(ctx, &pb.CreateBackfillRequest{Backfill: bf})
	require.NoError(t, err)
	require.Equal(t, int64(1), createdBf.Generation)

	createdBf.SearchFields.StringArgs["key"] = "val"

	orig := &anypb.Any{Value: []byte("test")}
	val, err := ptypes.MarshalAny(orig)
	require.NoError(t, err)

	// Create a different Backfill, but with the same ID
	// Pass different time
	bf2 := &pb.Backfill{
		CreateTime: ptypes.TimestampNow(),
		Id:         createdBf.Id,
		SearchFields: &pb.SearchFields{
			StringArgs: map[string]string{
				"key":    "val",
				"search": "me",
			},
		},
		Generation: 42,
		Extensions: map[string]*any.Any{"key": val},
	}
	updatedBf, err := om.Frontend().UpdateBackfill(ctx, &pb.UpdateBackfillRequest{Backfill: bf2})
	require.NoError(t, err)
	require.Equal(t, int64(2), updatedBf.Generation)

	// No changes to CreateTime
	require.Equal(t, createdBf.CreateTime.GetNanos(), updatedBf.CreateTime.GetNanos())

	get, err := om.Frontend().GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: createdBf.Id})
	require.NoError(t, err)
	require.Equal(t, bf2.SearchFields.StringArgs, get.SearchFields.StringArgs)

	unpacked := &anypb.Any{}
	err = ptypes.UnmarshalAny(get.Extensions["key"], unpacked)
	require.NoError(t, err)

	require.Equal(t, unpacked.Value, orig.Value)
	_, err = om.Frontend().DeleteBackfill(ctx, &pb.DeleteBackfillRequest{BackfillId: createdBf.Id})
	require.NoError(t, err)

	get, err = om.Frontend().GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: createdBf.Id})
	require.Error(t, err, fmt.Sprintf("Backfill id: %s not found", bf.Id))
	require.Nil(t, get)
}

// TestAcknowledgeBackfill checks that tickets got assigned
// to the same Connection as it is provided in AcknowledgeBackfill request
func TestAcknowledgeBackfill(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	bf := &pb.Backfill{SearchFields: &pb.SearchFields{
		StringArgs: map[string]string{
			"search": "me",
		},
	}}
	createdBf, err := om.Frontend().CreateBackfill(ctx, &pb.CreateBackfillRequest{Backfill: bf})
	require.NoError(t, err)
	require.Equal(t, int64(1), createdBf.Generation)

	ticketIDs := createMatchWithBackfill(ctx, om, createdBf, t)

	conn := "127.0.0.1:4242"
	getResp, err := om.Frontend().AcknowledgeBackfill(ctx, &pb.AcknowledgeBackfillRequest{
		BackfillId: createdBf.Id,
		Assignment: &pb.Assignment{
			Connection: conn,
			Extensions: map[string]*any.Any{
				"evaluation_input": mustAny(&pb.DefaultEvaluationCriteria{
					Score: 10,
				}),
			},
		},
	})
	require.NotNil(t, getResp)
	require.NotNil(t, getResp.Backfill)
	require.NotNil(t, getResp.Tickets)
	require.Equal(t, len(ticketIDs), len(getResp.Tickets))
	require.NoError(t, err)

	respTicketIds := make([]string, len(getResp.Tickets))

	for _, rt := range getResp.Tickets {
		respTicketIds = append(respTicketIds, rt.Id)
	}

	for _, v := range ticketIDs {
		ticket, err := om.Frontend().GetTicket(ctx, &pb.GetTicketRequest{TicketId: v})
		require.NoError(t, err)
		require.Contains(t, respTicketIds, ticket.Id)
		require.NotNil(t, ticket.Assignment)
		require.Equal(t, conn, ticket.Assignment.Connection)
	}
}

// TestAcknowledgeBackfillDeletedTicket checks that ticket deletion
// does not block other tickets in backfill from being assigned
func TestAcknowledgeBackfillDeletedTicket(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	bf := &pb.Backfill{SearchFields: &pb.SearchFields{
		StringArgs: map[string]string{
			"search": "me",
		},
	},
	}
	createdBf, err := om.Frontend().CreateBackfill(ctx, &pb.CreateBackfillRequest{Backfill: bf})
	require.NoError(t, err)
	require.Equal(t, int64(1), createdBf.Generation)

	ticketIDs := createMatchWithBackfill(ctx, om, createdBf, t)

	// Delete 1st ticket
	om.Frontend().DeleteTicket(ctx, &pb.DeleteTicketRequest{TicketId: ticketIDs[0]})
	conn := "127.0.0.1:4242"
	getResp, err := om.Frontend().AcknowledgeBackfill(ctx, &pb.AcknowledgeBackfillRequest{BackfillId: createdBf.Id, Assignment: &pb.Assignment{Connection: conn, Extensions: map[string]*any.Any{
		"evaluation_input": mustAny(&pb.DefaultEvaluationCriteria{
			Score: 10,
		}),
	}}})
	require.NotNil(t, getResp)
	require.NotNil(t, getResp.Backfill)
	require.NotNil(t, getResp.Tickets)
	require.NoError(t, err)

	// Check that an error on 1st ticket assignment does not change 2nd ticket assignment
	ticket, err := om.Frontend().GetTicket(ctx, &pb.GetTicketRequest{TicketId: ticketIDs[1]})
	require.NoError(t, err)
	require.NotNil(t, ticket.Assignment)
	require.Equal(t, conn, ticket.Assignment.Connection)
}

func createMatchWithBackfill(ctx context.Context, om *om, b *pb.Backfill, t *testing.T) []string {
	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{
		Ticket: &pb.Ticket{
			SearchFields: &pb.SearchFields{
				StringArgs: map[string]string{
					"field": "value",
				},
			},
		},
	})
	require.NoError(t, err)
	t2, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{
		Ticket: &pb.Ticket{
			SearchFields: &pb.SearchFields{
				StringArgs: map[string]string{
					"field": "value",
				},
			},
		},
	})
	require.NoError(t, err)

	m := &pb.Match{
		MatchId:  "1",
		Tickets:  []*pb.Ticket{t1, t2},
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
	require.NoError(t, err)

	resp, err := stream.Recv()
	require.NoError(t, err)
	bfID := resp.Match.Backfill.Id

	resp, err = stream.Recv()
	require.Nil(t, resp)
	require.Error(t, err)
	require.Equal(t, io.EOF.Error(), err.Error())

	actual, err := om.Frontend().GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: bfID})
	require.NoError(t, err)
	require.NotNil(t, actual)

	// Backfill Generation should be autoincremented if updated after MMF run
	// default Generation is 1 if not set (on Create)
	b.Generation++
	b.Id = actual.Id
	b.CreateTime = actual.CreateTime
	require.True(t, proto.Equal(b, actual))
	return []string{t1.Id, t2.Id}
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
	require.NoError(t, err)

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

	{
		stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
			Config:  om.MMFConfigGRPC(),
			Profile: &pb.MatchProfile{},
		})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.NotNil(t, resp)
		require.NoError(t, err)

		resp, err = stream.Recv()
		require.Nil(t, resp)
		require.Error(t, err)
		require.Equal(t, io.EOF.Error(), err.Error())
	}
	{
		stream, err := om.query.QueryBackfills(ctx, &pb.QueryBackfillsRequest{
			Pool: &pb.Pool{
				StringEqualsFilters: []*pb.StringEqualsFilter{
					{
						StringArg: "field",
						Value:     "value",
					},
				},
			},
		})
		require.NoError(t, err)

		resp, err := stream.Recv()
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Backfills, 1)

		b.Id = resp.Backfills[0].Id
		b.Generation = 1
		b.CreateTime = resp.Backfills[0].CreateTime
		require.True(t, proto.Equal(b, resp.Backfills[0]))

		resp, err = stream.Recv()
		require.Nil(t, resp)
		require.Error(t, err)
		require.Equal(t, io.EOF.Error(), err.Error())
	}
	{
		stream, err := om.Query().QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{
			StringEqualsFilters: []*pb.StringEqualsFilter{{StringArg: "field", Value: "value"}},
		}})
		require.NoError(t, err)

		_, err = stream.Recv()
		require.Error(t, err)
		require.Equal(t, io.EOF.Error(), err.Error())
	}
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
	require.NoError(t, err)

	b, err := om.Frontend().CreateBackfill(ctx, &pb.CreateBackfillRequest{Backfill: &pb.Backfill{
		SearchFields: &pb.SearchFields{
			StringArgs: map[string]string{
				"field": "value",
			},
		},
	}})
	require.NoError(t, err)

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

	{
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
		require.Error(t, err)
		require.Equal(t, io.EOF.Error(), err.Error())
	}
	{
		stream, err := om.query.QueryBackfills(ctx, &pb.QueryBackfillsRequest{
			Pool: &pb.Pool{
				StringEqualsFilters: []*pb.StringEqualsFilter{
					{
						StringArg: "field1",
						Value:     "value1",
					},
				},
			},
		})
		require.NoError(t, err)

		resp, err := stream.Recv()
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Backfills, 1)

		// Backfill Generation should be autoincremented
		b.Generation++
		require.True(t, proto.Equal(b, resp.Backfills[0]))

		resp, err = stream.Recv()
		require.Nil(t, resp)
		require.Error(t, err)
		require.Equal(t, io.EOF.Error(), err.Error())
	}
	{
		stream, err := om.Query().QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{
			StringEqualsFilters: []*pb.StringEqualsFilter{{StringArg: "field", Value: "value"}},
		}})
		require.NoError(t, err)

		_, err = stream.Recv()
		require.Error(t, err)
		require.Equal(t, io.EOF.Error(), err.Error())
	}
}

func TestBackfillGenerationMismatch(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)
	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{
		SearchFields: &pb.SearchFields{
			StringArgs: map[string]string{
				"field": "value",
			},
		},
	}})
	require.NoError(t, err)

	b, err := om.Frontend().CreateBackfill(ctx, &pb.CreateBackfillRequest{Backfill: &pb.Backfill{}})
	require.NoError(t, err)

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

	{
		stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
			Config:  om.MMFConfigGRPC(),
			Profile: &pb.MatchProfile{},
		})
		require.NoError(t, err)

		resp, err := stream.Recv()
		require.Nil(t, resp)
		require.Error(t, err)
		require.Equal(t, io.EOF.Error(), err.Error())
	}
	{
		stream, err := om.Query().QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{
			StringEqualsFilters: []*pb.StringEqualsFilter{{StringArg: "field", Value: "value"}},
		}})
		require.NoError(t, err)

		resp, err := stream.Recv()
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Tickets, 1)
		require.Equal(t, t1.Id, resp.Tickets[0].Id)

		resp, err = stream.Recv()
		require.Nil(t, resp)
		require.Error(t, err)
		require.Equal(t, io.EOF.Error(), err.Error())
	}
}

func TestCleanUpExpiredBackfills(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.NoError(t, err)
	require.NotNil(t, t1)

	b1, err := om.Frontend().CreateBackfill(ctx, &pb.CreateBackfillRequest{Backfill: &pb.Backfill{
		SearchFields: &pb.SearchFields{
			StringArgs: map[string]string{
				"search": "me",
			},
		}}})
	require.NoError(t, err)
	require.NotNil(t, b1)

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		return nil
	})

	// wait until backfill is expired, then try to get it
	time.Sleep(pendingReleaseTimeout * 2)

	// statestore.CleanupBackfills is called at the end of each synchronizer cycle after fetch matches call, so expired backfill will be removed
	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.NoError(t, err)
	resp, err := stream.Recv()
	require.Nil(t, resp)
	require.Error(t, err)
	require.Equal(t, io.EOF.Error(), err.Error())

	// call FetchMatches twice in order to give backfills time to be completely cleaned up
	stream, err = om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.NoError(t, err)

	resp, err = stream.Recv()
	require.Nil(t, resp)
	require.Error(t, err)
	require.Equal(t, io.EOF.Error(), err.Error())

	_, err = om.Frontend().GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: b1.Id})
	require.Error(t, err)
	require.Equal(t, fmt.Sprintf("rpc error: code = NotFound desc = Backfill id: %s not found", b1.Id), err.Error())
}

func TestBackfillSkipNotfoundError(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.NoError(t, err)
	require.NotNil(t, t1)

	b1, err := om.Frontend().CreateBackfill(ctx, &pb.CreateBackfillRequest{Backfill: &pb.Backfill{
		SearchFields: &pb.SearchFields{
			StringArgs: map[string]string{
				"search": "me",
			},
		}}})
	require.NoError(t, err)
	require.NotNil(t, b1)

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

		out <- p.MatchId
		return nil
	})

	// Delete Backfill to cause NotFound error
	_, err = om.Frontend().DeleteBackfill(ctx, &pb.DeleteBackfillRequest{BackfillId: b1.Id})
	require.NoError(t, err)

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.NoError(t, err)
	resp, err := stream.Recv()
	require.Nil(t, resp)
	require.Error(t, err)
	require.Equal(t, io.EOF.Error(), err.Error())

	_, err = om.Frontend().GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: b1.Id})
	require.Error(t, err)
	require.Equal(t, fmt.Sprintf("rpc error: code = NotFound desc = Backfill id: %s not found", b1.Id), err.Error())
}

func mustAny(m proto.Message) *any.Any {
	result, err := ptypes.MarshalAny(m)
	if err != nil {
		panic(err)
	}
	return result
}

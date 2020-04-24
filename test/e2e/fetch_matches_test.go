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
	"io"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
)

func TestHappyPath(t *testing.T) {
	ctx := context.Background()
	om := e2e.New(t)

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)
	t2, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	m := &pb.Match{
		MatchId: "1",
		Tickets: []*pb.Ticket{t1, t2},
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

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	resp, err := stream.Recv()
	require.Nil(t, err)
	require.True(t, proto.Equal(m, resp.Match))

	resp, err = stream.Recv()
	require.Equal(t, err, io.EOF)
	require.Nil(t, resp)
}

func TestMatchFunctionMatchCollision(t *testing.T) {
	ctx := context.Background()
	om := e2e.New(t)

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	t2, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		out <- &pb.Match{
			MatchId: "1",
			Tickets: []*pb.Ticket{t1},
		}
		out <- &pb.Match{
			MatchId: "1",
			Tickets: []*pb.Ticket{t2},
		}
		return nil
	})

	om.SetEvaluator(evaluatorRejectAll)

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	resp, err := stream.Recv()
	require.Contains(t, err.Error(), "found duplicate matchID")
	require.Nil(t, resp)
}

func TestEvaluatorMatchCollision(t *testing.T) {
	ctx := context.Background()
	om := e2e.New(t)

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		out <- &pb.Match{
			MatchId: "1",
			Tickets: []*pb.Ticket{t1},
		}
		return nil
	})

	om.SetEvaluator(evaluatorAcceptAll)

	s1, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	// Flow first match through before starting second mmf, so the second mmf
	// is the one which gets the collision error.
	_, err = s1.Recv()
	require.Nil(t, err)

	s2, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	resp, err := s2.Recv()
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "found duplicate matchID")

	resp, err = s1.Recv()
	require.Contains(t, err.Error(), "found duplicate matchID")
	require.Nil(t, resp)
}

func TestEvaluatorReturnInvalidId(t *testing.T) {
}

func TestEvaluatorReturnDuplicateMatchId(t *testing.T) {

}

func TestMatchWithNoTickets(t *testing.T) {

}

func evaluatorRejectAll(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
	for range in {
	}
	return nil
}

func evaluatorAcceptAll(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
	for m := range in {
		out <- m.MatchId
	}
	return nil
}

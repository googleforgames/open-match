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
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/pkg/pb"
)

// TestHappyPath does a simple test of sucessfully creating a match with two tickets.
func TestHappyPath(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

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

// TestMatchFunctionMatchCollision covers two matches with the same id coming
// from the same MMF generates an error to the fetch matches call.
func TestMatchFunctionMatchCollision(t *testing.T) {
	// TODO: another MMF in same cycle doesn't get error?
	ctx := context.Background()
	om := newOM(t)

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
	require.Contains(t, err.Error(), "MatchMakingFunction returned same match_id twice: \"1\"")
	require.Nil(t, resp)
}

// TestSynchronizerMatchCollision covers two different MMFs generating matches
// with the same id, and causing all fetch match calls to fail with an error
// indicating this occured.
func TestSynchronizerMatchCollision(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		out <- &pb.Match{
			MatchId: "1",
			Tickets: []*pb.Ticket{t1},
		}
		return nil
	})

	timesEvaluatorCalled := 0

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		timesEvaluatorCalled++
		require.Equal(t, 1, timesEvaluatorCalled)
		for m := range in {
			out <- m.MatchId
		}
		return nil
	})

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
	require.Contains(t, err.Error(), "Multiple match functions used same match_id: \"1\"")

	resp, err = s1.Recv()
	require.Contains(t, err.Error(), "Multiple match functions used same match_id: \"1\"")
	require.Nil(t, resp)
}

// TestEvaluatorReturnInvalidId covers the evaluator returning an ID which does
// not correspond to any match passed to it.
func TestEvaluatorReturnInvalidId(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		out <- "1"
		return nil
	})

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	resp, err := stream.Recv()
	require.Contains(t, err.Error(), "evaluator returned match_id \"1\" which does not correspond to its any match in its input")
	require.Nil(t, resp)
}

// TestEvaluatorReturnDuplicateMatchId covers the evaluator returning the same
// match id twice, which causes an error for fetch match callers.
func TestEvaluatorReturnDuplicateMatchId(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

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
		out <- m.MatchId
		return nil
	})

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	_, err = stream.Recv()

	// May recieve up to one match
	if err == nil {
		_, err = stream.Recv()
	}
	require.Contains(t, err.Error(), "evaluator returned same match_id twice: \"1\"")
}

// TestMatchWithNoTickets covers that it is valid to create a match with no
// tickets specified.  This is a questionable use case, but it works currently
// so it probably shouldn't be changed without significant justification.
func TestMatchWithNoTickets(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	m := &pb.Match{
		MatchId: "1",
		Tickets: nil,
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

// TestEvaluatorError covers an evaluator returning an error message, and
// ensuring that the error message is returned to the fetch matches call.
func TestEvaluatorError(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		return errors.New("my custom error")
	})

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	resp, err := stream.Recv()

	require.Contains(t, err.Error(), "my custom error")
	require.Nil(t, resp)
}

// TestEvaluatorError covers an MMF returning an error message, and ensuring
// that the error message is returned to the fetch matches call.
func TestMMFError(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		return errors.New("my custom error")
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		_, ok := <-in
		require.False(t, ok)
		return nil
	})

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	resp, err := stream.Recv()

	require.Contains(t, err.Error(), "my custom error")
	require.Nil(t, resp)
}

// TestNoMatches covers that returning no matches is acceptable.
func TestNoMatches(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		_, ok := <-in
		require.False(t, ok)

		return nil
	})

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	resp, err := stream.Recv()
	require.Equal(t, err, io.EOF)
	require.Nil(t, resp)
}

// TestNoMatches covers missing the profile field on fetch matches.
func TestNoProfile(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: nil,
	})
	require.Nil(t, err)

	resp, err := stream.Recv()
	require.Equal(t, ".profile is required", status.Convert(err).Message())
	require.Equal(t, codes.InvalidArgument, status.Convert(err).Code())
	require.Nil(t, resp)
}

// TestNoConfig covers missing the config field on fetch matches.
func TestNoConfig(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  nil,
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	resp, err := stream.Recv()
	require.Equal(t, ".config is required", status.Convert(err).Message())
	require.Equal(t, codes.InvalidArgument, status.Convert(err).Code())
	require.Nil(t, resp)
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

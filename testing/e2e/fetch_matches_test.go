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
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"open-match.dev/open-match/pkg/matchfunction"
	"open-match.dev/open-match/pkg/pb"
)

// TestHappyPath does a simple test of successfully creating a match with two tickets.
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
	require.Error(t, err)
	require.Equal(t, err.Error(), io.EOF.Error())
	require.Nil(t, resp)
}

// TestMatchFunctionMatchCollision covers two matches with the same id coming
// from the same MMF generates an error to the fetch matches call.  Also ensures
// another function running in the same cycle does not experience an error.
func TestMatchFunctionMatchCollision(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	t2, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	// Both mmf runs wait for the other to start before sending results, to
	// ensure they run in the same cycle.
	errorMMFStarted := make(chan struct{})
	successMMFStarted := make(chan struct{})

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		switch profile.Name {
		case "error":
			close(errorMMFStarted)
			<-successMMFStarted
			out <- &pb.Match{
				MatchId: "1",
				Tickets: []*pb.Ticket{t1},
			}
			out <- &pb.Match{
				MatchId: "1",
				Tickets: []*pb.Ticket{t2},
			}
		case "success":
			close(successMMFStarted)
			<-errorMMFStarted
			out <- &pb.Match{
				MatchId: "3",
				Tickets: []*pb.Ticket{t2},
			}
		default:
			panic("Unknown profile!")
		}
		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		for m := range in {
			if m.MatchId == "3" {
				out <- "3"
			}
		}
		return nil
	})

	startTime := time.Now()

	sError, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config: om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{
			Name: "error",
		},
	})
	require.Nil(t, err)

	sSuccess, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config: om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{
			Name: "success",
		},
	})
	require.Nil(t, err)

	resp, err := sError.Recv()
	require.Contains(t, err.Error(), "MatchMakingFunction returned same match_id twice: \"1\"")
	require.Nil(t, resp)

	resp, err = sSuccess.Recv()
	require.Nil(t, err)
	require.True(t, proto.Equal(t2, resp.Match.Tickets[0]))

	require.True(t, time.Since(startTime) < registrationInterval, "%s", time.Since(startTime))

	resp, err = sSuccess.Recv()
	require.Error(t, err)
	require.Equal(t, err.Error(), io.EOF.Error())
	require.Nil(t, resp)
}

// TestSynchronizerMatchCollision covers two different MMFs generating matches
// with the same id, and causing all fetch match calls to fail with an error
// indicating this occurred.
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
	require.Contains(t, err.Error(), "multiple match functions used same match_id: \"1\"")

	resp, err = s1.Recv()
	require.Contains(t, err.Error(), "multiple match functions used same match_id: \"1\"")
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

	// May receive up to one match
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
	require.Error(t, err)
	require.Equal(t, err.Error(), io.EOF.Error())
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
	require.Error(t, err)
	require.Equal(t, err.Error(), io.EOF.Error())
	require.Nil(t, resp)
}

// TestNoProfile covers missing the profile field on fetch matches.
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

// TestCancel covers a fetch matches call canceling also causing mmf and
// evaluator to cancel.
func TestCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	om := newOM(t)

	startTime := time.Now()

	wgStarted := sync.WaitGroup{}
	wgStarted.Add(2)
	wgFinished := sync.WaitGroup{}
	wgFinished.Add(2)

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		wgStarted.Done()
		<-ctx.Done()
		require.Equal(t, ctx.Err(), context.Canceled)
		wgFinished.Done()
		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		wgStarted.Done()
		<-ctx.Done()
		require.Equal(t, ctx.Err(), context.Canceled)
		wgFinished.Done()
		return nil
	})

	_, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	wgStarted.Wait()
	cancel()
	wgFinished.Wait()

	// The evaluator is only canceled after the registration window completes.
	require.True(t, time.Since(startTime) > registrationInterval, "%s", time.Since(startTime))
}

// TestStreaming covers that matches can stream through the mmf, evaluator, and
// return to the fetch matches call.  At no point are all matches accumulated
// and then passed on.  This keeps things efficiently moving.
func TestStreaming(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	wg := sync.WaitGroup{}
	wg.Add(1)

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)
	t2, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	m1 := &pb.Match{
		MatchId: "1",
		Tickets: []*pb.Ticket{t1},
	}
	m2 := &pb.Match{
		MatchId: "2",
		Tickets: []*pb.Ticket{t2},
	}

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		out <- m1
		wg.Wait()
		out <- m2
		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		<-in
		out <- "1"
		wg.Wait()
		<-in
		out <- "2"

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
	require.Nil(t, err)
	require.True(t, proto.Equal(m1, resp.Match))

	wg.Done()

	resp, err = stream.Recv()
	require.Nil(t, err)
	require.True(t, proto.Equal(m2, resp.Match))

	resp, err = stream.Recv()
	require.Error(t, err)
	require.Equal(t, err.Error(), io.EOF.Error())
	require.Nil(t, resp)
}

// TestRegistrationWindow covers a synchronization cycle waiting for the
// registration window before closing the evaluator.  However it also does not
// wait until the proposal window has closed if the mmfs have already returned.
func TestRegistrationWindow(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	startTime := time.Now()

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		_, ok := <-in
		require.False(t, ok)

		require.True(t, time.Since(startTime) > registrationInterval, "%s", time.Since(startTime))
		return nil
	})

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	resp, err := stream.Recv()
	require.Error(t, err)
	require.Equal(t, err.Error(), io.EOF.Error())
	require.Nil(t, resp)
	require.True(t, time.Since(startTime) > registrationInterval, "%s", time.Since(startTime))
}

// TestProposalWindowClose covers that a long running match function will get
// canceled so that the cycle can complete.
func TestProposalWindowClose(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	startTime := time.Now()

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		<-ctx.Done()
		require.Equal(t, ctx.Err(), context.Canceled)
		require.True(t, time.Since(startTime) > registrationInterval+proposalCollectionInterval, "%s", time.Since(startTime))
		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		_, ok := <-in
		require.False(t, ok)
		require.True(t, time.Since(startTime) > registrationInterval+proposalCollectionInterval, "%s", time.Since(startTime))
		return nil
	})

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	resp, err := stream.Recv()
	require.Contains(t, err.Error(), "match function ran longer than proposal window, canceling")
	require.Nil(t, resp)

	require.True(t, time.Since(startTime) > registrationInterval+proposalCollectionInterval, "%s", time.Since(startTime))
}

// TestMultipleFetchCalls covers multiple fetch matches calls running in the
// same cycle, using the same evaluator call, and having matches routed back to
// the correct caller.
func TestMultipleFetchCalls(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)
	t2, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	m1 := &pb.Match{
		MatchId: "1",
		Tickets: []*pb.Ticket{t1},
	}
	m2 := &pb.Match{
		MatchId: "2",
		Tickets: []*pb.Ticket{t2},
	}

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		switch profile.Name {
		case "one":
			out <- m1
		case "two":
			out <- m2
		default:
			return errors.New("Unknown profile")
		}

		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		ids := []string{}
		for m := range in {
			ids = append(ids, m.MatchId)
		}
		require.ElementsMatch(t, ids, []string{"1", "2"})
		for _, id := range ids {
			out <- id
		}
		return nil
	})

	s1, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config: om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{
			Name: "one",
		},
	})
	require.Nil(t, err)

	s2, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config: om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{
			Name: "two",
		},
	})
	require.Nil(t, err)

	resp, err := s1.Recv()
	require.Nil(t, err)
	require.True(t, proto.Equal(m1, resp.Match))

	resp, err = s1.Recv()
	require.Error(t, err)
	require.Equal(t, err.Error(), io.EOF.Error())
	require.Nil(t, resp)

	resp, err = s2.Recv()
	require.Nil(t, err)
	require.True(t, proto.Equal(m2, resp.Match))

	resp, err = s2.Recv()
	require.Error(t, err)
	require.Equal(t, err.Error(), io.EOF.Error())
	require.Nil(t, resp)
}

// TestSlowBackendDoesntBlock covers that after the evaluator has returned, a
// new cycle can start despite and slow fetch matches caller.  Additionally, it
// confirms that the tickets are marked as pending, so the second cycle won't be
// using any of the tickets returned by the first cycle.
func TestSlowBackendDoesntBlock(t *testing.T) {
	ctx := context.Background()
	om := newOM(t)

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)
	t2, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	m1 := &pb.Match{
		MatchId: "1",
		Tickets: []*pb.Ticket{t1},
	}
	m2 := &pb.Match{
		MatchId: "2",
		Tickets: []*pb.Ticket{t2},
	}

	evaluatorDone := make(chan struct{})

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		pool, mmfErr := matchfunction.QueryPool(ctx, om.Query(), &pb.Pool{})
		require.Nil(t, mmfErr)
		ids := []string{}
		for _, t := range pool {
			ids = append(ids, t.Id)
		}

		switch profile.Name {
		case "one":
			require.ElementsMatch(t, ids, []string{t1.Id, t2.Id})
			out <- m1
		case "two":
			require.ElementsMatch(t, ids, []string{t2.Id})
			out <- m2
		default:
			return errors.New("Unknown profile")
		}

		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		m := <-in
		_, ok := <-in
		require.False(t, ok)
		out <- m.MatchId

		evaluatorDone <- struct{}{}
		return nil
	})

	s1, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config: om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{
			Name: "one",
		},
	})
	require.Nil(t, err)

	<-evaluatorDone

	s2, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config: om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{
			Name: "two",
		},
	})
	require.Nil(t, err)

	<-evaluatorDone

	resp, err := s2.Recv()
	require.Nil(t, err)
	require.True(t, proto.Equal(m2, resp.Match))

	resp, err = s2.Recv()
	require.Error(t, err)
	require.Equal(t, err.Error(), io.EOF.Error())
	require.Nil(t, resp)

	resp, err = s1.Recv()
	require.Nil(t, err)
	require.True(t, proto.Equal(m1, resp.Match))

	resp, err = s1.Recv()
	require.Error(t, err)
	require.Equal(t, err.Error(), io.EOF.Error())
	require.Nil(t, resp)
}

// TestHTTPMMF covers calling the MMF with http config instead of gRPC.
func TestHTTPMMF(t *testing.T) {
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
		Config:  om.MMFConfigHTTP(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	resp, err := stream.Recv()
	require.Nil(t, err)
	require.True(t, proto.Equal(m, resp.Match))

	resp, err = stream.Recv()
	require.Error(t, err)
	require.Equal(t, err.Error(), io.EOF.Error())
	require.Nil(t, resp)
}

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

package synchronizer

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/ipb"
	"open-match.dev/open-match/internal/statestore"
	"open-match.dev/open-match/pkg/pb"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.synchronizer",
	})
)

// Matches flow through channels in the synchronizer.  Channel variable names
// are used to be consistent between function calls to help track everything.

// Streams from multiple GRPC calls of matches are combined on a single channel.
// These matches are sent to the evaluator, then the tickets are added to the
// ignore list.  Finally the matches are returned to the calling stream.

// receive from backend                  | Synchronize
//  -> m1c ->
// close incoming when done or timed out | newCutoffSender
//   -> m2c ->
// remember return channel m6c for match | fanInFanOut
//   -> m3c -> (buffered)
// send to evaluator                     | wrapEvaluator
//   -> m4c -> (buffered)
// add tickets to ignore list            | addMatchesToIgnoreList
//   -> m5c ->
// fan out to origin synchronize call    | fanInFanOut
//   -> (Synchronize call specific ) m6c -> (buffered)
// return to backend                     | Synchronize

type synchronizerService struct {
	cfg   config.View
	store statestore.Service
	eval  evaluator

	synchronizeRegistration chan *registrationRequest

	// startCycle is a buffered channel for containing a single value.  The value
	// is present only when a cycle is not running.
	startCycle chan struct{}
}

func newSynchronizerService(cfg config.View, eval evaluator, store statestore.Service) *synchronizerService {
	s := &synchronizerService{
		cfg:   cfg,
		store: store,
		eval:  eval,

		synchronizeRegistration: make(chan *registrationRequest),
		startCycle:              make(chan struct{}, 1),
	}

	s.startCycle <- struct{}{}

	return s
}

func (s *synchronizerService) Synchronize(stream ipb.Synchronizer_SynchronizeServer) error {
	// Synchronize first registers against a cycle.  Then it creates two go
	// routines:
	// 1. Receive proposals from backend, send them to cycle.
	// 2. Receive matches and signals from cycle, send them to backend.

	registration := s.register(stream.Context())
	m6cBuffer := bufferChannel(registration.m6c)
	defer func() {
		for range m6cBuffer {
		}
	}()

	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				if err != io.EOF {
					logger.WithFields(logrus.Fields{
						"error": err.Error(),
					}).Error("error streaming in synchronizer from backend")
				}
				registration.allM1cSent.Done()
				return
			}
			registration.m1c.send(mAndM6c{m: req.Proposal, m6c: registration.m6c})
		}
	}()

	err := stream.Send(&ipb.SynchronizeResponse{StartMmfs: true})
	if err != nil {
		return err
	}

	for {
		select {
		case matches, ok := <-m6cBuffer:
			if !ok {
				return nil
			}
			for _, match := range matches {
				err = stream.Send(&ipb.SynchronizeResponse{Match: match})
				if err != nil {
					logger.WithFields(logrus.Fields{
						"error": err.Error(),
					}).Error("error streaming match in synchronizer to backend")
					return err
				}
			}
		case <-registration.cancelMmfs:
			err = stream.Send(&ipb.SynchronizeResponse{CancelMmfs: true})
			if err != nil {
				logger.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("error streaming mmf cancel in synchronizer to backend")
				return err
			}
		case <-stream.Context().Done():
			logger.WithFields(logrus.Fields{
				"error": stream.Context().Err().Error(),
			}).Error("error streaming in synchronizer to backend: context is done")
			return stream.Context().Err()
		case <-registration.cycleCtx.Done():
			return registration.cycleCtx.Err()
		}
	}

}

///////////////////////////////////////
///////////////////////////////////////

// Registration of a Synchronize call for a cycle does the following:
// - Sends a registration request, starting a cycle if none is running.
// - The cycle creates the registration.
// - The registration is sent back to the origin synchronize call on channel as
//     part of the sychronize request.

type registrationRequest struct {
	resp chan *registration
	ctx  context.Context
}

type registration struct {
	m1c        *cutoffSender
	allM1cSent *sync.WaitGroup
	m6c        chan *pb.Match
	cancelMmfs chan struct{}
	cycleCtx   context.Context
}

func (s synchronizerService) register(ctx context.Context) *registration {
	req := &registrationRequest{
		resp: make(chan *registration),
		ctx:  ctx,
	}
	for {
		select {
		case s.synchronizeRegistration <- req:
			return <-req.resp
		case <-s.startCycle:
			go func() {
				s.runCycle()
				s.startCycle <- struct{}{}
			}()
		}
	}
}

///////////////////////////////////////
///////////////////////////////////////

func (s *synchronizerService) runCycle() {
	/////////////////////////////////////// Initialize cycle
	ctx, cancel := withCancelCause(context.Background())

	m2c := make(chan mAndM6c)
	m3c := make(chan *pb.Match)
	m4c := make(chan *pb.Match)
	m5c := make(chan *pb.Match)

	m1c := newCutoffSender(m2c)
	// m6c, unlike other channels, is specific to a synchronize call.  There are
	// multiple values in a given cycle.

	var allM1cSent sync.WaitGroup

	registrations := []*registration{}
	callingCtx := []context.Context{}
	closedOnCycleEnd := make(chan struct{})

	go func() {
		fanInFanOut(m2c, m3c, m5c)
		// Close response channels after all responses have been sent.
		for _, r := range registrations {
			close(r.m6c)
		}
	}()
	go s.wrapEvaluator(ctx, cancel, bufferChannel(m3c), m4c)
	go func() {
		s.addMatchesToIgnoreList(ctx, cancel, bufferChannel(m4c), m5c)
		// Wait for ignore list, but not all matches returned, the next cycle
		// can start now.
		close(closedOnCycleEnd)
	}()

	/////////////////////////////////////// Run Registration Period
	closeRegistration := time.After(s.registrationInterval())
Registration:
	for {
		select {
		case req := <-s.synchronizeRegistration:
			allM1cSent.Add(1)
			callingCtx = append(callingCtx, req.ctx)
			r := &registration{
				m1c:        m1c,
				m6c:        make(chan *pb.Match),
				cancelMmfs: make(chan struct{}, 1),
				cycleCtx:   ctx,
				allM1cSent: &allM1cSent,
			}
			registrations = append(registrations, r)
			req.resp <- r
		case <-closeRegistration:
			break Registration
		}
	}
	/////////////////////////////////////// Wait for cycle completion.

	go func() {
		for _, ctx := range callingCtx {
			<-ctx.Done()
		}
		cancel(fmt.Errorf("canceled because all callers were done"))
	}()

	go func() {
		allM1cSent.Wait()
		m1c.cutoff()
	}()

	cancelProposalCollection := time.AfterFunc(s.proposalCollectionInterval(), func() {
		m1c.cutoff()
		for _, r := range registrations {
			r.cancelMmfs <- struct{}{}
		}
	})
	<-closedOnCycleEnd

	// Clean up in case it was never needed.
	cancelProposalCollection.Stop()
}

///////////////////////////////////////
///////////////////////////////////////

type mAndM6c struct {
	m   *pb.Match
	m6c chan *pb.Match
}

// fanInFanOut routes evaluated matches back to it's source synchronize call.
// Each incoming match is passed along with it's synchronize call's m6c channel.
// This channel is remembered in a map, and the match is passed to be evaluated.
// When a match returns from evaluation, it's ID is looked up in the map and the
// match is returned on that channel.
func fanInFanOut(m2c <-chan mAndM6c, m3c chan<- *pb.Match, m5c <-chan *pb.Match) {
	m6cMap := make(map[string]chan<- *pb.Match)

	defer func(m2c <-chan mAndM6c) {
		for range m2c {
		}
	}(m2c)

	for {
		select {
		case m2, ok := <-m2c:
			if ok {
				m6cMap[m2.m.GetMatchId()] = m2.m6c
				m3c <- m2.m
			} else {
				close(m3c)
				// No longer select on m2c
				m2c = nil
			}

		case m5, ok := <-m5c:
			if !ok {
				return
			}

			m6c, ok := m6cMap[m5.GetMatchId()]
			if ok {
				m6c <- m5
			} else {
				logger.WithFields(logrus.Fields{
					"matchId": m5.GetMatchId(),
				}).Error("Match ID from evaluator does not match any id sent to it.")
			}
		}
	}
}

///////////////////////////////////////
///////////////////////////////////////

type cutoffSender struct {
	m1c       chan<- mAndM6c
	m2c       chan<- mAndM6c
	closed    chan struct{}
	closeOnce sync.Once
}

// cutoffSender allows values to be passed on the provided channel until cutoff
// has been called.  This closed the provided channel.  Calls to send after
// cutoff work, but values are ignored.
func newCutoffSender(m2c chan<- mAndM6c) *cutoffSender {
	m1c := make(chan mAndM6c)
	c := &cutoffSender{
		m1c:    m1c,
		m2c:    m2c,
		closed: make(chan struct{}),
	}

	go func() {
		defer close(m2c)

		for {
			select {
			case <-c.closed:
				return
			case match := <-m1c:
				m2c <- match
			}
		}
	}()
	return c
}

// send passes the value on the channel if still open, otherwise does nothing.
func (c *cutoffSender) send(match mAndM6c) {
	select {
	case <-c.closed:
	case c.m1c <- match:
	}
}

// cutoff closes m2c.  Safe to call from multiple go routines.
func (c *cutoffSender) cutoff() {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
}

///////////////////////////////////////
///////////////////////////////////////

// Calls the evaluator with the matches.
func (s *synchronizerService) wrapEvaluator(ctx context.Context, cancel cancelErrFunc, m3c <-chan []*pb.Match, m4c chan<- *pb.Match) {

	// TODO: Stream through the request.

	proposalList := []*pb.Match{}
	for matches := range m3c {
		proposalList = append(proposalList, matches...)
	}

	matchList, err := s.eval.evaluate(ctx, proposalList)
	if err == nil {
		for _, m := range matchList {
			m4c <- m
		}
	} else {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("error calling evaluator, canceling cycle")
		cancel(fmt.Errorf("error calling evaluator: %w", err))
	}
	close(m4c)
}

///////////////////////////////////////
///////////////////////////////////////

// Calls statestore to add all of the tickets returned by the evaluator to the
// ignorelist.  If it partially fails for whatever reason (not all tickets will
// nessisarily be in the same call), only the matches which can be safely
// returned to the Synchronize calls are.
func (s *synchronizerService) addMatchesToIgnoreList(ctx context.Context, cancel cancelErrFunc, m4c <-chan []*pb.Match, m5c chan<- *pb.Match) {
	totalMatches := 0
	successfulMatches := 0
	var lastErr error
	for matches := range m4c {
		ids := []string{}
		for _, match := range matches {
			for _, ticket := range match.GetTickets() {
				ids = append(ids, ticket.GetId())
			}
		}

		err := s.store.AddTicketsToIgnoreList(ctx, ids)

		totalMatches += len(matches)
		if err == nil {
			successfulMatches += len(matches)
		} else {
			lastErr = err
		}

		for _, match := range matches {
			m5c <- match
		}
	}

	if lastErr != nil {
		logger.WithFields(logrus.Fields{
			"error":             lastErr.Error(),
			"totalMatches":      totalMatches,
			"successfulMatches": successfulMatches,
		}).Error("some or all matches were not successfully added to the ignore list, failed matches dropped")

		if successfulMatches == 0 {
			cancel(fmt.Errorf("no matches successfully added to the ignore list.  Last error: %w", lastErr))
		}
	}
	close(m5c)
}

///////////////////////////////////////
///////////////////////////////////////

func (s *synchronizerService) registrationInterval() time.Duration {
	const (
		name            = "synchronizer.registrationIntervalMs"
		defaultInterval = time.Second
	)

	if !s.cfg.IsSet(name) {
		return defaultInterval
	}

	return s.cfg.GetDuration(name)
}

func (s *synchronizerService) proposalCollectionInterval() time.Duration {
	const (
		name            = "synchronizer.proposalCollectionIntervalMs"
		defaultInterval = 10 * time.Second
	)

	if !s.cfg.IsSet(name) {
		return defaultInterval
	}

	return s.cfg.GetDuration(name)
}

///////////////////////////////////////
///////////////////////////////////////

// bufferChannel collects matches from the input, and sends
// slice of matches on the output.  It never (for long) blocks
// the input channel, always appending to the slice which will
// next be used for output.  Used before external calls, so that
// network won't back up internal processing.
func bufferChannel(in chan *pb.Match) chan []*pb.Match {
	out := make(chan []*pb.Match)
	go func() {
		var a []*pb.Match

	outerLoop:
		for {
			m, ok := <-in
			if !ok {
				break outerLoop
			}
			a = []*pb.Match{m}

			for len(a) > 0 {
				select {
				case m, ok := <-in:
					if !ok {
						break outerLoop
					}
					a = append(a, m)
				case out <- a:
					a = nil
				}
			}
		}
		if len(a) > 0 {
			out <- a
		}
		close(out)
	}()
	return out
}

///////////////////////////////////////
///////////////////////////////////////

// getMatchIds returns all of the match_id values on the slice of matches.
func getMatchIds(matches []*pb.Match) []string {
	var result []string
	for _, m := range matches {
		result = append(result, m.GetMatchId())
	}
	return result
}

///////////////////////////////////////
///////////////////////////////////////

// withCancelCause returns a copy of parent with a new Done channel. The
// returned context's Done channel is closed when the returned cancel function
// is called or when the parent context's Done channel is closed, whichever
// happens first.  Unlike the conext package's WithCancel, the cancel func takes
// an error, and will return that error on subsequent calls to Err().
func withCancelCause(parent context.Context) (context.Context, cancelErrFunc) {
	parent, cancel := context.WithCancel(parent)

	ctx := &contextWithCancelCause{
		Context: parent,
	}

	return ctx, func(err error) {
		ctx.m.Lock()
		defer ctx.m.Unlock()

		if err == nil && parent.Err() == nil {
			ctx.err = err
		}
		cancel()
	}
}

type cancelErrFunc func(err error)

type contextWithCancelCause struct {
	context.Context
	m   sync.Mutex
	err error
}

func (ctx *contextWithCancelCause) Err() error {
	ctx.m.Lock()
	defer ctx.m.Unlock()
	if ctx.err == nil {
		return ctx.Context.Err()
	}
	return ctx.err
}

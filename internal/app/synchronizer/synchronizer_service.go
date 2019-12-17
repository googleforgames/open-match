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
// are unused to be consistant between function calls to help track everything.

// Streams from multiple GRPC calls of matches are combined on a single channel.
// These matches are sent to the evaluator, then the tickets are added to the
// ignore list.  Finally the matches are returned to the calling stream.

// This is best read from bottom to top, as both Synchronize and runCycle combine
// components further in the file to create the desired functionality.

// receive from backend                       | Synchronize
//  -> m1c ->
// collect from multiple synchronize calls    | combineWithCutoff
//   -> m2c ->
// remember return channel (m TODO) for match | fanInFanOut
//   -> m3c -> (buffered)
// send to evaluator                          | wrapEvaluator
//   -> m4c -> (buffered)
// add tickets to ignore list                 | addMatchesToIgnoreList
//   -> m5c ->
// fan out to origin synchronize call         | fanInFanOut
//   -> m6c -> (buffered)
// return to backend                          | Synchronize

type synchronizerService struct {
	cfg       config.View
	store     statestore.Service
	evaluator evaluator

	synchronizeRegistration chan *registrationRequest

	// startCycle is a buffered channel for containing a single value.  The value
	// is present only when a cycle is not running.
	startCycle chan struct{}
}

func newSynchronizerService(cfg config.View, evaluator evaluator, store statestore.Service) *synchronizerService {
	s := &synchronizerService{
		cfg:       cfg,
		store:     store,
		evaluator: evaluator,

		synchronizeRegistration: make(chan *registrationRequest),
		startCycle:              make(chan struct{}, 1),
	}

	s.startCycle <- struct{}{}

	return s
}

func (s *synchronizerService) Synchronize(stream ipb.Synchronizer_SynchronizeServer) error {
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
					}).Error("Error streaming in synchronizer from backend.")
				}
				registration.m1cDone <- struct{}{}
				return
			}
			select {
			case registration.m1c <- mAndM6c{m: req.Proposal, m6c: registration.m6c}:
			case <-registration.closedOnMmfCancel:
			}
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
					}).Error("Error streaming match in synchronizer to backend.")
					return err
				}
			}
		case <-registration.cancelMmfs:
			err = stream.Send(&ipb.SynchronizeResponse{CancelMmfs: true})
			if err != nil {
				logger.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("Error streaming mmf cancel in synchronizer to backend.")
				return err
			}
		case <-stream.Context().Done():
			logger.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Error streaming in synchronizer to backend: context is done")
			// TODO: LOG ERROR
			return stream.Context().Err()
		case <-registration.cycleCtx.Done():
			return registration.cycleCtx.Err()
		}
	}

}

///////////////////////////////////////
///////////////////////////////////////

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

func (s *synchronizerService) runCycle() {
	ctx, cancel := WithCancelCause(context.Background())

	m1c := make(chan mAndM6c)
	m2c := make(chan mAndM6c)
	m3c := make(chan *pb.Match)
	m4c := make(chan *pb.Match)
	m5c := make(chan *pb.Match)
	// m6c is per Synchronize call.

	// Value passed indicates a Synchronize call has finished sending values.  It can't
	// close the channel because other calls may still be using it.
	m1cDone := make(chan struct{})
	// Value passed indicates a new Synchronize call has joined the cycle.
	newRegistration := make(chan struct{})
	// Value passed indicates that no new Synchronize calls will join the cycle.
	registrationDone := make(chan struct{})

	registrations := []*registration{}
	callingCtx := []context.Context{}
	closedOnMmfCancel := make(chan struct{})
	closedOnCycleEnd := make(chan struct{})

	go combineWithCutoff(m1c, m2c, registrationDone, newRegistration, m1cDone, closedOnMmfCancel)
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
		close(closedOnCycleEnd)
	}()

	closeRegistration := time.After(s.registrationInterval())
Registration:
	for {
		select {
		case req := <-s.synchronizeRegistration:
			callingCtx = append(callingCtx, req.ctx)
			r := &registration{
				m1c:               m1c,
				m1cDone:           m1cDone,
				m6c:               make(chan *pb.Match),
				cancelMmfs:        make(chan struct{}, 1),
				closedOnMmfCancel: closedOnMmfCancel,
				cycleCtx:          ctx,
			}
			newRegistration <- struct{}{}
			registrations = append(registrations, r)
			req.resp <- r
		case <-closeRegistration:
			break Registration
		}
	}
	registrationDone <- struct{}{}
	go cancelWhenCallersAreDone(callingCtx, cancel)

	cancelProposalCollection := time.AfterFunc(s.proposalCollectionInterval(), func() {
		close(closedOnMmfCancel)
		for _, r := range registrations {
			r.cancelMmfs <- struct{}{}
		}
	})
	<-closedOnCycleEnd

	// Clean up in case it was never needed.
	cancelProposalCollection.Stop()
}

type registrationRequest struct {
	resp chan *registration
	ctx  context.Context
}

type registration struct {
	m1c               chan mAndM6c
	m1cDone           chan struct{}
	m6c               chan *pb.Match
	cancelMmfs        chan struct{}
	closedOnMmfCancel chan struct{}
	cycleCtx          context.Context
}

///////////////////////////////////////
///////////////////////////////////////

type mAndM6c struct {
	m   *pb.Match
	m6c chan *pb.Match
}

// For incoming mathces, remembers the channel the match needs to be returned to
// if it passes the synchronizer.
func fanInFanOut(m2c <-chan mAndM6c, m3c chan<- *pb.Match, m5c <-chan *pb.Match) {
	m6cMap := make(map[string]chan<- *pb.Match)

	defer func() {
		for range m2c {
		}
	}()

loop:
	for {
		select {
		case m2, ok := <-m2c:
			if !ok {
				close(m3c)
				break loop
			}
			m6cMap[m2.m.GetMatchId()] = m2.m6c
			m3c <- m2.m

		case m5, ok := <-m5c:
			if !ok {
				return
			}

			m6cMap[m5.GetMatchId()] <- m5
		}
	}

	for m5 := range m5c {
		m6cMap[m5.GetMatchId()] <- m5
	}
}

///////////////////////////////////////
///////////////////////////////////////

func combineWithCutoff(m1c <-chan mAndM6c, m2c chan<- mAndM6c, registrationDone, newRegistration, m1cDone, closedOnMmfCancel chan struct{}) {
	defer close(m2c)

	registrationOpen := true
	openSenders := 0

	for {
		if !registrationOpen && openSenders == 0 {
			return
		}

		select {
		case <-newRegistration:
			openSenders++
		case <-m1cDone:
			openSenders--
		case <-closedOnMmfCancel:
			return
		case <-registrationDone:
			registrationOpen = false
		case match := <-m1c:
			m2c <- match
		}
	}
}

///////////////////////////////////////
///////////////////////////////////////

// Calls the evaluator with the matches.
func (s *synchronizerService) wrapEvaluator(ctx context.Context, cancel CancelErrFunc, m3c <-chan []*pb.Match, m4c chan<- *pb.Match) {

	// TODO: Stream through the request.

	proposalList := []*pb.Match{}
	for matches := range m3c {
		proposalList = append(proposalList, matches...)
	}

	matchList, err := s.evaluator.evaluate(ctx, proposalList)
	if err == nil {
		for _, m := range matchList {
			m4c <- m
		}
	} else {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error calling evaluator. Canceling cycle.")
		cancel(fmt.Errorf("Error calling evaluator: %w", err))
	}
	close(m4c)
}

///////////////////////////////////////
///////////////////////////////////////

// Calls statestore to add all of the tickets returned by the evaluator to the
// ignorelist.  If it partially fails for whatever reason (not all tickets will
// nessisarily be in the same call), only the matches which can be safely
// returned to the Synchronize calls are.
func (s *synchronizerService) addMatchesToIgnoreList(ctx context.Context, cancel CancelErrFunc, m4c <-chan []*pb.Match, m5c chan<- *pb.Match) {
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
		}).Error("Some or all matches were not successfully added to the ignore list, failed matches dropped.")

		if successfulMatches == 0 {
			cancel(fmt.Errorf("No matches successfully added to the ignore list.  Last error: %w", lastErr))
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

// WithCancelCause returns a copy of parent with a new Done channel. The
// returned context's Done channel is closed when the returned cancel function
// is called or when the parent context's Done channel is closed, whichever
// happens first.  Unlike the conext package's WithCancel, the cancel func takes
// an error, and will return that error on subsiquent calls to Err().
func WithCancelCause(parent context.Context) (context.Context, CancelErrFunc) {
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

type CancelErrFunc func(err error)

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

///////////////////////////////////////
///////////////////////////////////////

// Waits for all contexts to be Done, then calls cancel.
func cancelWhenCallersAreDone(ctxs []context.Context, cancel CancelErrFunc) {
	for _, ctx := range ctxs {
		<-ctx.Done()
	}
	cancel(fmt.Errorf("Canceled because all callers were done."))
}

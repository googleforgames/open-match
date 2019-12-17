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
	matchesBuffer := bufferChannel(registration.m6c)
	defer func() {
		for range matchesBuffer {
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
		case matches, ok := <-matchesBuffer:
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
		case <-registration.unification.Context().Done():
			return registration.unification.Err()
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

	unification := newCallUnification()

	m1c := make(chan mAndM6c)
	m2c := make(chan mAndM6c)
	m3c := make(chan *pb.Match)
	m4c := make(chan *pb.Match)
	m5c := make(chan *pb.Match)
	// m6c is per Synchronize call.

	m1cDone := make(chan struct{})

	newRegistration := make(chan struct{})
	registrationDone := make(chan struct{})

	registrations := []*registration{}
	closedOnMmfCancel := make(chan struct{})
	matchesAddedToIgnoreList := make(chan struct{})

	go combineWithCutoff(unification, m1c, m2c, registrationDone, newRegistration, m1cDone, closedOnMmfCancel)
	go func() {
		fanInFanOut(unification, m2c, m3c, m5c)
		for _, r := range registrations {
			close(r.m6c)
		}
	}()
	go s.wrapEvaluator(unification, bufferChannel(m3c), m4c)
	go func() {
		s.addMatchesToIgnoreList(unification, bufferChannel(m4c), m5c)
		close(matchesAddedToIgnoreList)
	}()

	closeRegistration := time.After(s.registrationInterval())
Registration:
	for {
		select {
		case req := <-s.synchronizeRegistration:
			unification.Attatch(req.ctx)
			r := &registration{
				m1c:               m1c,
				m1cDone:           m1cDone,
				m6c:               make(chan *pb.Match),
				cancelMmfs:        make(chan struct{}, 1),
				closedOnMmfCancel: closedOnMmfCancel,
				unification:       unification,
			}
			registrations = append(registrations, r)
			req.resp <- r
		case <-closeRegistration:
			break Registration
		}
	}
	registrationDone <- struct{}{}
	unification.CancelIfAttatchedDone()

	cancelProposalCollection := time.AfterFunc(s.proposalCollectionInterval(), func() {
		close(closedOnMmfCancel)
		for _, r := range registrations {
			r.cancelMmfs <- struct{}{}
		}
	})

	<-matchesAddedToIgnoreList

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
	unification       *callUnification
}

///////////////////////////////////////
///////////////////////////////////////

type mAndM6c struct {
	m   *pb.Match
	m6c chan *pb.Match
}

func fanInFanOut(unification *callUnification, m2c <-chan mAndM6c, m3c chan<- *pb.Match, m5c <-chan *pb.Match) {

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

func combineWithCutoff(unification *callUnification, m1c <-chan mAndM6c, m2c chan<- mAndM6c, registrationDone, newRegistration, m1cDone, closedOnMmfCancel chan struct{}) {
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
		case <-unification.Context().Done():
			return
		}
	}
}

///////////////////////////////////////
///////////////////////////////////////

func (s *synchronizerService) wrapEvaluator(unification *callUnification, m3c <-chan []*pb.Match, m4c chan<- *pb.Match) {

	// TODO: Stream through the request.

	proposalList := []*pb.Match{}
	for matches := range m3c {
		proposalList = append(proposalList, matches...)
	}

	matchList, err := s.evaluator.evaluate(context.Background(), proposalList)
	if err == nil {
		for _, m := range matchList {
			m4c <- m
		}
	} else {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error calling evaluator. Canceling cycle.")
		unification.CancelWithError(fmt.Errorf("Error calling evaluator: %w", err))
	}
	close(m4c)
}

///////////////////////////////////////
///////////////////////////////////////

func (s *synchronizerService) addMatchesToIgnoreList(unification *callUnification, m4c <-chan []*pb.Match, m5c chan<- *pb.Match) {
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

		err := s.store.AddTicketsToIgnoreList(unification.Context(), ids)

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
	}
	if totalMatches > 0 && successfulMatches == 0 {
		unification.CancelWithError(fmt.Errorf("No matches successfully added to the ignore list.  Last error: %w", lastErr))
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

func getMatchIds(matches []*pb.Match) []string {
	var result []string
	for _, m := range matches {
		result = append(result, m.GetMatchId())
	}
	return result
}

///////////////////////////////////////
///////////////////////////////////////

type callUnification struct {
	ctx                   context.Context
	cancelWithErr         chan error
	err                   error
	attatch               chan context.Context
	cancelIfAttatchedDone chan struct{}
}

func newCallUnification() *callUnification {
	ctx, cancel := context.WithCancel(context.Background())

	c := &callUnification{
		ctx:                   ctx,
		cancelWithErr:         make(chan error),
		err:                   nil,
		attatch:               make(chan context.Context),
		cancelIfAttatchedDone: make(chan struct{}),
	}
	go func() {
		defer cancel()
		neverSends := (<-chan struct{})(make(chan struct{}))
		sourceContexts := []context.Context{}
		cancelIfAttatchedDone := false
		for {
			nextSourceContext := neverSends
			if cancelIfAttatchedDone {
				if len(sourceContexts) == 0 {
					c.err = fmt.Errorf("All callers have canceled, so canceling.")
					return
				}
				nextSourceContext = sourceContexts[0].Done()
			}
			select {
			case <-nextSourceContext:
				sourceContexts = sourceContexts[1:]
			case aCtx := <-c.attatch:
				sourceContexts = append(sourceContexts, aCtx)
			case c.err = <-c.cancelWithErr:
				cancel()
			case <-ctx.Done():
				c.err = ctx.Err()
				return
			case <-c.cancelIfAttatchedDone:
				cancelIfAttatchedDone = true
			}
		}
	}()

	return c
}

func (c *callUnification) Attatch(ctx context.Context) {
	select {
	case c.attatch <- ctx:
	case <-c.ctx.Done():
	}
}

func (c *callUnification) CancelWithError(err error) {
	select {
	case <-c.ctx.Done():
	case c.cancelWithErr <- err:
	}
}

func (c *callUnification) Err() error {
	select {
	case <-c.ctx.Done():
		return c.err
	default:
		return nil
	}
}

func (c *callUnification) Context() context.Context {
	return c.ctx
}

func (c *callUnification) CancelIfAttatchedDone() {
	select {
	case c.cancelIfAttatchedDone <- struct{}{}:
	case <-c.ctx.Done():
	}
}

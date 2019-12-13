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

type synchronizerService struct {
	cfg       config.View
	store     statestore.Service
	evaluator evaluator

	synchronizeRegistration chan chan *registration
	// startCycle is a buffered channel for containing a single value.  The value
	// is present only when a cycle is not running.
	startCycle chan struct{}
}

func newSynchronizerService(cfg config.View, evaluator evaluator, store statestore.Service) *synchronizerService {
	s := &synchronizerService{
		cfg:       cfg,
		store:     store,
		evaluator: evaluator,

		synchronizeRegistration: make(chan chan *registration),
		startCycle:              make(chan struct{}, 1),
	}

	s.startCycle <- struct{}{}

	return s
}

// Matches flow through channels in the synchronizer.  Match variables are
// named by how many steps of processing have taken place.

// receive from backend                       | Synchronize
//   m1 -> m1c
// collect from multiple synchronize calls    | foobarRenameMe
//   m2 -> m2c
// remember return channel (m TODO) for match | fanInFanOut
//   m3 -> m3c (buffered)
// send to evaluator                          | wrapEvaluator
//   m4 -> m4c (buffered)
// add tickets to ignore list                 | addMatchesToIgnoreList
//   m5 -> m5c
// fan out to origin synchronize call         | fanInFanOut
//   m6 -> m6c (buffered)
// return to backend                          | Synchronize

func (s *synchronizerService) Synchronize(stream ipb.Synchronizer_SynchronizeServer) error {
	//logger.Warning("============= Synchronize start")
	//defer logger.Warning("============= Synchronize end")

	registration := s.register()
	defer func() {
		for range registration.m6c {
		}
	}()

	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				if err != io.EOF {
					// logger.

					// panic(err)
					/////////////////////////todo log error
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
		case match, ok := <-registration.m6c:
			if !ok {
				return nil
			}
			err = stream.Send(&ipb.SynchronizeResponse{Match: match})
			if err != nil {
				// TODO LOG ERROR
				return err
			}
		case <-registration.cancelMmfs:
			err = stream.Send(&ipb.SynchronizeResponse{CancelMmfs: true})
			if err != nil {
				// TODO LOG ERROR
				return err
			}
		case <-stream.Context().Done():
			// TODO: LOG ERROR
			return stream.Context().Err()
		}
	}

}

///////////////////////////////////////
///////////////////////////////////////

func (s synchronizerService) register() *registration {
	resp := make(chan *registration)
	for {
		select {
		case s.synchronizeRegistration <- resp:
			return <-resp
		case <-s.startCycle:
			go func() {
				s.runCycle()
				s.startCycle <- struct{}{}
			}()
		}
	}
}

func (s *synchronizerService) runCycle() {
	//logger.Warning("============= runCycle start")
	//defer logger.Warning("============= runCycle end")

	m1c := make(chan mAndM6c)
	m2c := make(chan mAndM6c)
	m3c := make(chan *pb.Match)
	m4c := make(chan *pb.Match)
	m5c := make(chan *pb.Match)
	// m6c is per Synchronize call.

	m1cDone := make(chan struct{})

	newRegistration := make(chan struct{})
	closeRegistration := time.After(s.registrationInterval())
	registrationDone := make(chan struct{})

	registrations := []*registration{}
	closedOnMmfCancel := make(chan struct{})
	matchesAddedToIgnoreList := make(chan struct{})

	go foobarRenameMe(m1c, m2c, registrationDone, newRegistration, m1cDone, closedOnMmfCancel)
	go func() {
		fanInFanOut(m2c, m3c, m5c)
		for _, r := range registrations {
			close(r.m6c) // TODO when switching to buffered, needs to close the buffer input, not buffer output
		}
	}()
	go s.wrapEvaluator(bufferChannel(m3c), m4c)
	go func() {
		s.addMatchesToIgnoreList(bufferChannel(m4c), m5c)
		close(matchesAddedToIgnoreList)
	}()

Registration:
	for {
		select {
		case c := <-s.synchronizeRegistration:
			r := &registration{
				m1c:               m1c,
				m1cDone:           m1cDone,
				m6c:               make(chan *pb.Match),
				cancelMmfs:        make(chan struct{}, 1),
				closedOnMmfCancel: closedOnMmfCancel,
			}
			registrations = append(registrations, r)
			c <- r
		case <-closeRegistration:
			break Registration
		}
	}
	registrationDone <- struct{}{}

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

type registration struct {
	m1c               chan mAndM6c
	m1cDone           chan struct{}
	m6c               chan *pb.Match
	cancelMmfs        chan struct{}
	closedOnMmfCancel chan struct{}
}

///////////////////////////////////////
///////////////////////////////////////

type mAndM6c struct {
	m   *pb.Match
	m6c chan *pb.Match
}

func fanInFanOut(m2c <-chan mAndM6c, m3c chan<- *pb.Match, m5c <-chan *pb.Match) {
	//logger.Warning("============= fanInFanOut start")
	//defer logger.Warning("============= fanInFanOut end")

	m6cMap := make(map[string]chan<- *pb.Match)

	defer func() {
		for range m2c {
		}
	}()

	for {
		select {
		case m2, ok := <-m2c:
			if !ok {
				close(m3c)
				return
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
}

///////////////////////////////////////
///////////////////////////////////////

func foobarRenameMe(m1c <-chan mAndM6c, m2c chan<- mAndM6c, registrationDone, newRegistration, m1cDone, closedOnMmfCancel chan struct{}) {
	//logger.Warning("============= foobarRenameMe start")
	//defer logger.Warning("============= foobarRenameMe end")
	registrationOpen := true
	openSenders := 0

	for {
		if !registrationOpen && openSenders == 0 {
			close(m2c)
			return
		}

		select {
		case <-newRegistration:
			openSenders++
		case <-m1cDone:
			openSenders--
		case <-closedOnMmfCancel:
			close(m2c)
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

func (s *synchronizerService) wrapEvaluator(proposals chan []*pb.Match, matches chan *pb.Match) {
	//logger.Warning("============= wrapEvaluator start")
	//defer logger.Warning("============= wrapEvaluator end")

	// TODO: Stream through the request.

	proposalList := []*pb.Match{}
	for p := range proposals {
		proposalList = append(proposalList, p...)
	}

	matchList, err := s.evaluator.evaluate(context.Background(), proposalList)
	if err != nil {
		logger.Error(err)
		// panic(err)
		///TODO: DO SOMETHING SENSIBLE
	}
	for _, m := range matchList {
		matches <- m
	}
	close(matches)
}

///////////////////////////////////////
///////////////////////////////////////

func (s *synchronizerService) addMatchesToIgnoreList(m4c <-chan []*pb.Match, m5c chan<- *pb.Match) {
	//logger.Warning("============= addMatchesToIgnoreList start")
	//defer logger.Warning("============= addMatchesToIgnoreList end")
	for m4s := range m4c {
		ids := []string{}
		for _, m4 := range m4s {
			for _, ticket := range m4.GetTickets() {
				ids = append(ids, ticket.GetId())
			}
		}

		err := s.store.AddTicketsToIgnoreList(context.Background(), ids)
		if err != nil {
			panic(err) // TODO: DO SOMETHING SENSIBLE
		}

		for _, m4 := range m4s {
			m5c <- m4
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

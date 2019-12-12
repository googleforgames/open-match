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

	synchronizeRequests chan chan *matchRouterClient
	// startCycle is a buffered channel for containing a single value.  The value
	// is present only when a cycle is not running.
	startCycle chan struct{}
}

func newSynchronizerService(cfg config.View, evaluator evaluator, store statestore.Service) *synchronizerService {
	s := &synchronizerService{
		cfg:       cfg,
		store:     store,
		evaluator: evaluator,

		synchronizeRequests: make(chan chan *matchRouterClient),
		startCycle:          make(chan struct{}, 1),
	}

	s.startCycle <- struct{}{}

	return s
}

// Matches flow through channels in the synchronizer.

//
// +-----------+ --> +------+     +---------+
// |synchronize|     |match | --> |evaluator|
// +-----------+ <-- |router|     +---------+
//                   |      |          |
// +-----------+ --> |      |          V
// |synchronize|     |      |     +--------------+
// +-----------+ <-- +------+ <-- |reserveMatches|
//                                +--------------+

func (s *synchronizerService) Synchronize(stream ipb.Synchronizer_SynchronizeServer) error {
	resp := make(chan *matchRouterClient)

joinOrStart:
	for {
		select {
		case s.synchronizeRequests <- resp:
			break joinOrStart
		case <-s.startCycle:
			go s.runCycle()
		}
	}

	mrc := <-resp
	defer mrc.DrainRecv()

	cancelMmfs := make(chan struct{})

	go func() {
		defer mrc.CloseSend()
		cancelSent := false

		for {
			req, err := stream.Recv()
			if err != nil {
				if err != io.EOF {
					// logger.

					// panic(err)
					/////////////////////////todo log error
				}
				return
			}
			if !mrc.Send(req.Proposal) && !cancelSent {
				cancelMmfs <- struct{}{}
				cancelSent = true
			}
		}
	}()

	err := stream.Send(&ipb.SynchronizeResponse{StartMmfs: true})
	if err != nil {
		return err
	}

	for {
		select {
		case match, ok := <-mrc.Recv():
			if !ok {
				return nil
			}
			err = stream.Send(&ipb.SynchronizeResponse{Match: match})
			if err != nil {
				// TODO LOG ERROR
				return err
			}
		case <-cancelMmfs:
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

func (s *synchronizerService) runCycle() {
	incomingProposals := make(chan *pb.Match)
	reservedMatches := make(chan *pb.Match)
	evaluatedMatches := make(chan *pb.Match)

	mr := newMatchRouter(incomingProposals, reservedMatches)
	go s.wrapEvaluator(incomingProposals, evaluatedMatches)
	matchesReserved := make(chan struct{})
	go func() {
		s.reserveMatches(evaluatedMatches, reservedMatches)
		close(matchesReserved)
	}()

	registrationDone := time.After(s.registrationInterval())

Registration:
	for {
		select {
		case r := <-s.synchronizeRequests:
			r <- mr.NewClient()
		case <-registrationDone:
			break Registration
		}
	}
	mr.CloseNewClients()

	cancelProposalCollection := time.AfterFunc(s.proposalCollectionInterval(), func() {
		mr.CloseProposals()
	})

	// Once matches have been reserved, next cycle can started.
	<-matchesReserved

	// Clean up in case it was never needed.
	cancelProposalCollection.Stop()
	s.startCycle <- struct{}{}
}

///////////////////////////////////////
///////////////////////////////////////

type matchRouter struct {
	// Channels sent by function calls on matchRouter, recieved by the
	// matchRouter's go routine.  Never closed, because closed channels are always
	// available to recieve in a select, and it's easier to only have one select.
	proposals      chan propose
	closeProposals chan struct{}
	newClient      chan chan *pb.Match
	closeNewClient chan struct{}
	senderClosed   chan struct{}

	// Closed by matchRouter's go routine, used to prevent clients being blocked
	// after the matchRouter's go routine returns.
	proposalsClosed chan struct{}
}

type matchRouterClient struct {
	mr         *matchRouter
	matches    chan *pb.Match
	sendClosed bool
}

type propose struct {
	proposal *pb.Match
	result   chan *pb.Match
}

func newMatchRouter(proposalsOut, matchesIn chan *pb.Match) *matchRouter {
	mr := &matchRouter{
		proposals:      make(chan propose),
		closeProposals: make(chan struct{}),
		newClient:      make(chan chan *pb.Match),
		closeNewClient: make(chan struct{}),
		senderClosed:   make(chan struct{}),

		proposalsClosed: make(chan struct{}),
	}

	go func() {
		activeCount := 0
		collectingClients := true
		collectingProposals := true
		origin := make(map[string]chan *pb.Match)
		clients := []chan *pb.Match{}
		for {
			select {
			case proposal := <-mr.proposals:
				origin[proposal.proposal.GetMatchId()] = proposal.result
				proposalsOut <- proposal.proposal
			case <-mr.closeProposals:
				if collectingProposals {
					close(proposalsOut)
					close(mr.proposalsClosed)
					collectingProposals = false
				}
			case mrcMatches := <-mr.newClient:
				activeCount++
				clients = append(clients, mrcMatches)
			case <-mr.closeNewClient:
				collectingClients = false
				if activeCount == 0 && collectingProposals {
					close(proposalsOut)
					close(mr.proposalsClosed)
					collectingProposals = false
				}
			case <-mr.senderClosed:
				activeCount--
				if activeCount == 0 && collectingProposals && !collectingClients {
					close(proposalsOut)
					close(mr.proposalsClosed)
					collectingProposals = false
				}
			case match, ok := <-matchesIn:
				if !ok {
					for _, c := range clients {
						close(c)
					}
					return
				}
				origin[match.GetMatchId()] <- match
			}
		}
	}()

	return mr
}

func (mr *matchRouter) NewClient() *matchRouterClient {
	mrc := &matchRouterClient{
		mr:      mr,
		matches: make(chan *pb.Match),
	}

	mr.newClient <- mrc.matches

	return mrc
}

func (mr *matchRouter) CloseNewClients() {
	mr.closeNewClient <- struct{}{}
}

func (mr *matchRouter) CloseProposals() {
	mr.closeProposals <- struct{}{}
}

func (mrc *matchRouterClient) Send(proposal *pb.Match) bool {
	select {
	case mrc.mr.proposals <- propose{proposal: proposal, result: mrc.matches}:
		return true
	case <-mrc.mr.proposalsClosed:
		return false
	}
}

func (mrc *matchRouterClient) CloseSend() {
	mrc.mr.senderClosed <- struct{}{}
}

// Channel MUST be drained.
func (mrc *matchRouterClient) Recv() chan *pb.Match {
	return mrc.matches
}

func (mrc *matchRouterClient) DrainRecv() {
	for range mrc.Recv() {
	}
}

///////////////////////////////////////
///////////////////////////////////////

func (s *synchronizerService) wrapEvaluator(proposals, matches chan *pb.Match) {
	// TODO: Stream through the request.

	proposalList := []*pb.Match{}
	for p := range proposals {
		proposalList = append(proposalList, p)
	}

	matchList, err := s.evaluator.evaluate(context.Background(), proposalList)
	if err != nil {
		panic(err)
		///TODO: DO SOMETHING SENSIBLE
	}
	for _, m := range matchList {
		matches <- m
	}
	close(matches)
}

///////////////////////////////////////
///////////////////////////////////////

func (s *synchronizerService) reserveMatches(unreserved, reserved chan *pb.Match) {
	buffered := bufferChannel(unreserved)

	for matches := range buffered {
		ids := []string{}
		for _, match := range matches {
			for _, ticket := range match.GetTickets() {
				ids = append(ids, ticket.GetId())
			}
		}

		err := s.store.AddTicketsToIgnoreList(context.Background(), ids)
		if err != nil {
			panic(err) // TODO: DO SOMETHING SENSIBLE
		}

		for _, match := range matches {
			reserved <- match
		}
	}
	close(reserved)
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

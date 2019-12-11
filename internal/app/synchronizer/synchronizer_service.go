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
	store               statestore.Service
	synchronizeRequests chan chan *matchRouterClient
	evaluator           evaluator
}

func newSynchronizerService(cfg config.View, evaluator evaluator, store statestore.Service) *synchronizerService {
	s := &synchronizerService{
		store:               store,
		synchronizeRequests: make(chan chan *matchRouterClient),
		evaluator:           evaluator,
	}

	s.runCycles(s.synchronizeRequests)

	return s
}

// Running components:
// - Request - recieving proposals
// - Request - sending matches
// - Handling cycles
// - Routing matches back to caller
// - Sending to evaluator
// - Marking tickets as pending.

func (s *synchronizerService) Synchronize(stream ipb.Synchronizer_SynchronizeServer) error {
	resp := make(chan *matchRouterClient)
	s.synchronizeRequests <- resp

	mrc := <-resp
	defer mrc.DrainRecv()

	go func() {
		defer mrc.CloseSend()
		for {
			req, err := stream.Recv()
			if err != nil {
				if err != io.EOF {
					/////////////////////////todo log error
				}
				return
			}
			mrc.Send(req.Proposal)
		}
	}()

	for {
		match, ok := <-mrc.Recv()
		if !ok {
			return nil
		}
		err := stream.Send(&ipb.SynchronizeResponse{Match: match})
		if err != nil {
			// TODO LOG ERROR
			return err
		}
	}

}

///////////////////////////////////////
///////////////////////////////////////

func (s *synchronizerService) runCycles(synchronizeRequests chan chan *matchRouterClient) {
	for {
		{
			firstRequest := <-synchronizeRequests
			go func() {
				synchronizeRequests <- firstRequest
			}()
		}

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

		registrationDone := time.After(time.Second) ////////////////// TODO READ FROM CONFIG

	Registration:
		for {
			select {
			case r := <-synchronizeRequests:
				r <- mr.NewClient()
			case <-registrationDone:
				break Registration
			}
		}
		mr.CloseNewClients()

		cancelProposalCollection := time.AfterFunc(time.Second, func() { ////////////////// TODO READ FROM CONFIG
			mr.CloseProposals()
		})

		// Once matches have been reserved, next cycle can started.
		<-matchesReserved

		// Clean up in case it was never needed.
		cancelProposalCollection.Stop()
	}
}

///////////////////////////////////////
///////////////////////////////////////

type matchRouter struct {
	proposals       chan propose
	proposalsClosed chan struct{}
	newClient       chan chan *pb.Match
	senderClosed    chan struct{}

	origin         map[string]chan *pb.Match
	clients        []chan *pb.Match
	clientsSending int
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
		proposals:       make(chan propose),
		proposalsClosed: make(chan struct{}),
		senderClosed:    make(chan struct{}),
		origin:          make(map[string]chan *pb.Match),
		newClient:       make(chan chan *pb.Match),
	}

	go func() {

	}()

	return mr
}

func (mr *matchRouter) NewClient() *matchRouterClient {
	mrc := &matchRouterClient{
		mr:         mr,
		matches:    make(chan *pb.Match),
		sendClosed: false,
	}

	return mrc
}

func (mr *matchRouter) CloseNewClients() {
	close(mr.newClient)
}

func (mr *matchRouter) CloseProposals() {
	close(mr.proposalsClosed)
}

func (mrc *matchRouterClient) Send(proposal *pb.Match) bool {
	if mrc.sendClosed {
		panic("Sending on closed matchRouterCleint")
	}
	select {
	case mrc.mr.proposals <- propose{proposal: proposal, result: mrc.matches}:
		return true
	case <-mrc.mr.proposalsClosed:
		return false
	}
}

func (mrc *matchRouterClient) CloseSend() {
	if mrc.sendClosed {
		panic("Closing a closed matchRouterCleint")
	}
	mrc.mr.senderClosed <- struct{}{}
	mrc.sendClosed = true
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

func (s *synchronizerService) reserveMatches(proposals, matches chan *pb.Match) {
	props := bufferChannel(proposals)
	loop := true
	for {
		matches := []*pb.Match{}
	gather:
		for ps := range props {
			ids := []string{}
			for _, match := range ps {
				for _, ticket := range match.GetTickets() {
					ids = append(ids, ticket.GetId())
				}
			}

			err := s.store.AddTicketsToIgnoreList(context.Background(), ids)
			if err != nil {
				panic(err) // TODO: DO SOMETHING SENSIBLE
			}
		}
	}
}

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

// // The service implementing the Synchronizer API that synchronizes the evaluation
// // of proposals from Match functions.
// type synchronizerService struct {
// 	store            statestore.Service
// 	synchronizeRequestss chan chan *cycle
// 	evaluator        evaluator
// }

// func newSynchronizerService(cfg config.View, evaluator evaluator, store statestore.Service) *synchronizerService {
// 	s := &synchronizerService{
// 		store:           store,
// 		evaluator:       evaluator,
// 		egisterRequests: make(chan chan *cycle),
// 	}

// 	go s.run()

// 	return s
// }

// func (s *synchronizerService) Synchronize(stream ipb.Synchronizer_Synchronize) error {
// 	results := make(chan *pb.Match)

// 	respChan := make(chan *cycle)
// 	s.registerRequests <- respChan
// 	c <- respChan

// 	defer func() {
// 		// Drain results to unblock cycle go routine.
// 		for {
// 			select {
// 			case <-c.allMatchesSent:
// 				return
// 			case <-results:
// 			}
// 		}
// 	}()

// 	go func() {
// 		for {
// 			req, err := stream.Recv()
// 			if err != nil {
// 				select {
// 				case mmfsDone <- struct{}{}:
// 				case <-c.proposalWindowClosed:
// 				}
// 				if err != io.EOF {
// 					/////////////////////////todo log error
// 				}
// 				return
// 			}
// 			select {
// 			case c.proposals <- proposal{req.Proposal, results}:
// 			case <-c.proposalWindowClosed:
// 			}
// 		}
// 	}()

// 	err := stream.Send(&ipb.SynchronizeResponse{
// 		MmfStartWindowOpen: ptype.DurationProto(time.Second * 1), // TODO: subtract now from registrationClose
// 		ProposalWindowOpen: ptype.DurationProto(time.Second * 1),
// 		Match:              nil,
// 	})

// 	select {
// 	case <-stream.Context().Done():
// 		return
// 	case match := <-results:
// 		err = stream.Send(match)
// 		if err != nil {
// 			/////////////////////////todo log error
// 			return
// 		}
// 	case <-c.allMatchesSent:
// 		return
// 	}
// }

// type proposal struct {
// 	proposal *pb.Match
// 	results  chan *pb.Match
// }

// func (s *synchronizerService) run() {
// 	for {
// 		c := cycle{
// 			proposalReturn:       map[string]chan *pb.Match{},
// 			mmfsDone:             make(chan struct{}),
// 			proposalWindowClosed: make(chan struct{}),
// 			allMatchesSent:       make(chan struct{}),
// 			proposals:            make(chan proposal),
// 			registered:           0,
// 		}

// 		evalSend, evalRecv := s.wrapEvaluatorCall()

// 		c.registration(s.registerRequests, evalSend)
// 		c.proposalCollection(evalSend)
// 		c.matchReturn(evalRecv)
// 	}
// }

// type cycle struct {
// 	registrationClose time.Time

// 	proposals            chan proposal
// 	proposalWindowClosed chan struct{}
// 	mmfsDone             chan struct{}
// 	proposalClose        time.Time

// 	proposalReturn map[string]chan *pb.Match
// 	allMatchesSent chan struct{}

// 	registered int
// }

// func (c *cycle) registration(registerRequests chan chan *cycle, evalSend chan *pb.Match) {
// 	{
// 		first := <-c.registerRequests
// 		go func() {
// 			s.registerRequests <- first
// 		}()
// 	}
// 	// TODO c.registrationClose =
// 	// TODO c.proposalClose =
// 	closeRegistration := time.After(time.Second)

// 	for {
// 		select {
// 		case closeRegistration:
// 			return
// 		case r := <-registerRequests:
// 			r <- c
// 			c.registered++
// 		case proposal <- c.proposals:
// 			evalSend <- proposal.proposal
// 			c.proposalReturn[proposal.proposal.MatchId] = proposal.results
// 		}
// 	}
// }

// func (c *cycle) proposalCollection(evalSend chan *pb.Match) {
// 	closeProposalCollection := time.After(time.Second)
// 	toSend := []*pb.Match{}
// 	defer func() {
// 		close(proposalWindowClosed)
// 		close(evalSend)
// 	}()

// 	for {
// 		select {
// 		case <-c.mmfsDone:
// 			registered--
// 			if registered <= 0 {
// 				return
// 			}
// 		case proposal <- c.proposals:
// 			evalSend <- proposal.proposal
// 			c.proposalReturn[proposal.proposal.MatchId] = proposal.results
// 		case <-closeProposalCollection:
// 			return
// 		}
// 	}
// }

// func (c *cycle) matchReturn(evalRecv chan *pb.Match) {
// 	defer func() {
// 		close(allMatchesSent)
// 	}()

// 	for match := range evalRecv {
// 		caller, ok := c.proposalReturn[match.MatchId]
// 		if ok {
// 			caller <- match
// 		} else {
// 			// TODO: Log error
// 		}
// 	}
// }

// func (s *synchronizerService) wrapEvaluatorCall() (evalSend, evalRecv chan *pb.Match) {
// 	evalSend = make(chan *pb.Match)
// 	evalRecv = make(chan *pb.Match)
// 	go func() {
// 		// TODO: Start streaming to evaluator before this point.
// 		toSend := []*pb.Match{}
// 		for proposal := range evalSend {
// 			toSend = append(toSend, proposal)
// 		}

// 		// TODO: Better context choice.
// 		matches, err := s.evaluator.evaluate(context.Background(), proposal)
// 		if err != nil {

// 			// Return error to caller somehow???
// 			return
// 		}
// 		for _, m := range matches {
// 			evalRecv <- m
// 		}
// 	}()

// 	return evalSend, evalRecv
// }

// // Register associates this request with the current synchronization cycle and
// // returns an identifier for this registration. The caller returns this
// // identifier back in the evaluation request. This enables synchronizer to
// // identify stale evaluation requests belonging to a prior cycle.
// func (s *synchronizerService) Register(ctx context.Context, req *ipb.RegisterRequest) (*ipb.RegisterResponse, error) {
// 	// Registration calls are only permitted if the synchronizer is idle or in
// 	// registration state. If the synchronizer is in any other state, the
// 	// registration call blocks on the idle condition. This condition is broadcasted
// 	// when the previous synchronization cycle is complete, waking up all the
// 	// blocked requests to make progress in the new cycle.
// 	s.state.stateMutex.Lock()
// 	for (s.state.status != statusIdle) && (s.state.status != statusRequestRegistration) {
// 		s.state.idleCond.Wait()
// 	}

// 	defer s.state.stateMutex.Unlock()
// 	if s.state.status == statusIdle {
// 		// After waking up, the first request that encounters the idle state changes
// 		// state to requestRegistration and initializes the state for this synchronization
// 		// cycle. Consequent registration requests bypass this initialization.
// 		logger.Info("Changing status from idle to requestRegistration")
// 		s.state.status = statusRequestRegistration
// 		s.state.resetCycleData()

// 		// This method triggers the goroutine that changes the state of the synchronizer
// 		// to proposalCollection after a configured time.
// 		go s.state.trackRegistrationWindow()
// 	}

// 	// Now that we are in requestRegistration state, allocate an id and initialize
// 	// request data for this request.
// 	id := xid.New().String()
// 	s.state.cycleData.idToRequestData[id] = &requestData{}
// 	logger.WithFields(logrus.Fields{
// 		"id": id,
// 	}).Info("Registered request for synchronization")
// 	return &ipb.RegisterResponse{Id: id}, nil
// }

// // EvaluateProposals accepts a list of proposals and a registration identifier
// // for this request. If the synchronization cycle to which the request was
// // registered is completed, this request fails otherwise the proposals are
// // added to the list of proposals to be evaluated in the current cycle. At the
// //  end of the cycle, the user defined evaluation method is triggered and the
// // matches accepted by it are returned as results.
// func (s *synchronizerService) EvaluateProposals(stream ipb.Synchronizer_EvaluateProposalsServer) error {
// 	ctx := stream.Context()
// 	id := util.GetSynchronizerContextID(ctx)
// 	if len(id) == 0 {
// 		return status.Errorf(codes.Internal, "synchronizer context id should be an non-empty string")
// 	}

// 	var proposals = []*pb.Match{}
// 	for {
// 		req, err := stream.Recv()
// 		if err == io.EOF {
// 			break
// 		}

// 		if err != nil {
// 			return err
// 		}

// 		proposals = append(proposals, req.GetMatch())
// 	}

// 	results, err := s.doEvaluateProposals(stream.Context(), proposals, id)
// 	if err != nil {
// 		return err
// 	}

// 	for _, result := range results {
// 		if err = stream.Send(&ipb.EvaluateProposalsResponse{Match: result}); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

// func (s *synchronizerService) doEvaluateProposals(ctx context.Context, proposals []*pb.Match, id string) ([]*pb.Match, error) {
// 	// pendingRequests keeps track of number of requests pending. This is incremented
// 	// in addProposals while holding the state mutex. The count should be decremented
// 	// only after this request completes.
// 	err := s.state.addProposals(id, proposals)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// If proposals were successfully added, then the pending request count has been
// 	// incremented and should only be reduced when this request is completed.
// 	defer s.state.cycleData.pendingRequests.Done()

// 	// fetchResults blocks till results are available. It does not change the
// 	// state of the synchronizer itself (since multiple concurrent of these can
// 	// be in progress). The evaluation routine tracks the completion of all of
// 	// these and then changes the state of the synchronizer to idle.
// 	results, err := s.state.fetchResults(id)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ids := []string{}
// 	for _, match := range results {
// 		for _, ticket := range match.GetTickets() {
// 			ids = append(ids, ticket.GetId())
// 		}
// 	}
// 	err = s.store.AddTicketsToIgnoreList(ctx, ids)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return results, nil
// }

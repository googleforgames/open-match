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
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	ipb "open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/statestore"
	"open-match.dev/open-match/pkg/pb"
)

type synchronizerState int

// The Synchronizer can be in one of the following states.
const (
	stateIdle synchronizerState = iota
	stateRequestRegistration
	stateProposalCollection
	stateEvaluation
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.synchronizer",
	})
)

// The requestData holds the state of each pending request for evaluation. The
// lifespan of this data is same as the lifespan of a request.
type requestData struct {
	proposals []*pb.Match // Proposals added by the request for evaluation.
	results   []*pb.Match // Results from evaluation of the proposals.
}

// synchronizerData holds the state for a synchronization cycle. This data is
// reset on every new synchronization cycle.
type synchronizerData struct {
	// idToRequestData holds a map of the id for each evaluation request to the data
	// associated with that request. It tracks ids issued during registration in the
	// current synchronization cycle and mantain context for the evaluation call from them.
	idToRequestData map[string]*requestData
	evalError       error
	pendingRequests sync.WaitGroup // Tracks the completion of all pending requests
	resultsReady    chan struct{}  // Signal completion of evaluation and availability of results.
}

// The service implementing the Synchronizer API that synchronizes the evaluation
// of proposals from Match functions.
type synchronizerService struct {
	cfg        config.View        // Open Match configuration
	cycleData  *synchronizerData  // State for the current synchronization cycle
	eval       evaluator          // Evaluation function to be triggered
	store      statestore.Service // A wrapper service that connects to the backend data storage service
	idleCond   *sync.Cond         // Signal any blocked registrations to proceed
	state      synchronizerState  // Current state of the Synchronizer
	stateMutex sync.Mutex         // Mutex to check on current state and take specific actions.
}

func newSynchronizerService(cfg config.View, eval evaluator, store statestore.Service) *synchronizerService {
	service := &synchronizerService{
		state: stateIdle,
		cfg:   cfg,
		eval:  eval,
		store: store,
		cycleData: &synchronizerData{
			idToRequestData: make(map[string]*requestData),
			resultsReady:    make(chan struct{}),
		},
	}

	service.idleCond = sync.NewCond(&service.stateMutex)
	return service
}

// Register associates this request with the current synchronization cycle and
// returns an identifier for this registration. The caller returns this
// identifier back in the evaluation request. This enables synchronizer to
// identify stale evaluation requests belonging to a prior cycle.
func (s *synchronizerService) Register(ctx context.Context, req *ipb.RegisterRequest) (*ipb.RegisterResponse, error) {
	// Registration calls are only permitted if the synchronizer is idle or in
	// registration state. If the synchronizer is in any other state, the
	// registration call blocks on the idle condition. This condition is broadcasted
	// when the previous synchronization cycle is complete, waking up all the
	// blocked requests to make progress in the new cycle.
	s.stateMutex.Lock()
	for (s.state != stateIdle) && (s.state != stateRequestRegistration) {
		s.idleCond.Wait()
	}

	defer s.stateMutex.Unlock()
	if s.state == stateIdle {
		// After waking up, the first request that encounters the idle state changes
		// state to requestRegistration and initializes the state for this synchronization
		// cycle. Consequent registration requests bypass this initialization.
		logger.Info("Changing state from idle to requestRegistration")
		s.state = stateRequestRegistration
		s.cycleData = &synchronizerData{
			idToRequestData: make(map[string]*requestData),
			resultsReady:    make(chan struct{}),
		}

		// This method triggers the goroutine that changes the state of the synchronizer
		// to proposalCollection after a configured time.
		go s.trackRegistrationWindow()
	}

	// Now that we are in requestRegistration state, allocate an id and initialize
	// request data for this request.
	id := xid.New().String()
	s.cycleData.idToRequestData[id] = &requestData{}
	logger.WithFields(logrus.Fields{
		"id": id,
	}).Info("Registered request for synchronization")
	return &ipb.RegisterResponse{Id: id}, nil
}

// EvaluateProposals accepts a list of proposals and a registration identifier
// for this request. If the synchronization cycle to which the request was
// registered is completed, this request fails otherwise the proposals are
// added to the list of proposals to be evaluated in the current cycle. At the
//  end of the cycle, the user defined evaluation method is triggered and the
// matches accepted by it are returned as results.
func (s *synchronizerService) EvaluateProposals(ctx context.Context, req *ipb.EvaluateProposalsRequest) (*ipb.EvaluateProposalsResponse, error) {
	logger.WithFields(logrus.Fields{
		"id":        req.GetId(),
		"proposals": getMatchIds(req.GetMatches()),
	}).Info("Received request to evaluate propsals")
	// pendingRequests keeps track of number of requests pending. This is incremented
	// in addProposals while holding the state mutex. The count should be decremented
	// only after this request completes.
	err := s.addProposals(req.Id, req.Matches)
	if err != nil {
		return nil, err
	}

	// If proposals were successfully added, then the pending request count has been
	// incremented and should only be reduced when this request is completed.
	defer s.cycleData.pendingRequests.Done()

	// fetchResults blocks till results are available. It does not change the
	// state of the synchronizer itself (since multiple concurrent of these can
	// be in progress). The evaluation routine tracks the completion of all of
	// these and then changes the state of the synchronizer to idle.
	results, err := s.fetchResults(req.Id)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for _, match := range results {
		for _, ticket := range match.GetTickets() {
			ids = append(ids, ticket.GetId())
		}
	}
	err = s.store.AddTicketsToIgnoreList(ctx, ids)
	if err != nil {
		return nil, err
	}

	return &ipb.EvaluateProposalsResponse{Matches: results}, nil
}

func (s *synchronizerService) addProposals(id string, proposals []*pb.Match) error {
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()

	if !s.canAcceptProposal() {
		return status.Error(codes.DeadlineExceeded, "synchronizer currently not accepting match proposals")
	}

	if _, ok := s.cycleData.idToRequestData[id]; !ok {
		return status.Error(codes.DeadlineExceeded, "request not present in the current synchronization cycle")
	}

	if len(s.cycleData.idToRequestData[id].proposals) > 0 {
		// We do not currently support submitting multiple separate proposals for a registered request.
		return status.Error(codes.Internal, "multiple proposal addition for a request not supported")
	}

	// Proposal can be added for this request so increment the pending requests
	s.cycleData.pendingRequests.Add(1)

	// Copy the proposals being added to the request data for the specified id.
	s.cycleData.idToRequestData[id].proposals = make([]*pb.Match, len(proposals))
	copy(s.cycleData.idToRequestData[id].proposals, proposals)
	return nil
}

func (s *synchronizerService) canAcceptProposal() bool {
	return s.state == stateProposalCollection || s.state == stateRequestRegistration
}

func (s *synchronizerService) fetchResults(id string) ([]*pb.Match, error) {
	t := time.Now()
	var results []*pb.Match

	// fetchMatches will block till evaluation is in progress. The resultsReady channel
	// is closed only when evaluation is complete. This is used to signal all waiting
	// requests that the results are ready.
	// TODO: Add a timeout in case results are not ready within a certain duration.
	<-s.cycleData.resultsReady
	if s.cycleData.evalError != nil {
		logger.WithError(s.cycleData.evalError).Errorf(
			"Evaluation failed for request %v", id)
		return nil, s.cycleData.evalError
	}

	results = make([]*pb.Match, len(s.cycleData.idToRequestData[id].results))
	copy(results, s.cycleData.idToRequestData[id].results)
	logger.WithFields(logrus.Fields{
		"id":      id,
		"results": getMatchIds(results),
		"elapsed": time.Since(t).String(),
	}).Trace("Evaluation results ready")
	return results, nil
}

// Evaluate aggregates all the proposals across requests and calls the user configured
// evaluator with these. Once evaluator returns, it copies results back in respective
// request data objects and then signals the completion of evaluation. It then waits
// for all the result processing (by fetchMatches) to be compelted before changing the
// synchronizer state to idle.
// NOTE: This method is always called while holding the synchronizer state mutex.
func (s *synchronizerService) Evaluate() {
	aggregateProposals := []*pb.Match{}
	proposalMap := make(map[string]string)
	for id, data := range s.cycleData.idToRequestData {
		aggregateProposals = append(aggregateProposals, data.proposals...)
		for _, m := range data.proposals {
			proposalMap[m.MatchId] = id
		}
	}

	logger.WithFields(logrus.Fields{
		"proposals": getMatchIds(aggregateProposals),
	}).Info("Requesting evaluation of proposals")
	results, err := s.eval.evaluate(aggregateProposals)
	if err != nil {
		// Evaluation failed. Set the error on the synchronization cycle data and
		// signal completion of evaluation. Do not process any partial results.
		s.cycleData.evalError = err
		logger.WithError(err).Errorf("Failed to evaluate proposals: %v", getMatchIds(aggregateProposals))
	} else {
		// Evaluation succeeded. Process all the results and add them to the results list
		// for the corresponding request data
		logger.WithFields(logrus.Fields{
			"results": getMatchIds(results),
		}).Info("Evaluation successfully returned results")
		for _, m := range results {
			cid, ok := proposalMap[m.MatchId]
			if !ok {
				// Evaluator returned a match that the synchronizer did not submit for evaluation. This
				// could happen if the match id was modified accidentally.
				logger.WithFields(logrus.Fields{
					"match": m.GetMatchId(),
				}).Warning("Result does not belong to any known requests")
				continue
			}

			s.cycleData.idToRequestData[cid].results = append(s.cycleData.idToRequestData[cid].results, m)
		}
	}

	// Signal completion of evaluation to all requests waiting for results.
	close(s.cycleData.resultsReady)

	// Wait for all fetchMatches to complete processing results and then set synchronizer
	// to idle state and signal any blocked registration requests to proceed.
	s.cycleData.pendingRequests.Wait()
	logger.Info("Changing state from evaluating to idle")
	s.state = stateIdle
	s.idleCond.Broadcast()
}

func (s *synchronizerService) trackRegistrationWindow() {
	time.Sleep(s.registrationInterval())
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()
	logger.Info("Changing state from requestRegistration to proposalCollection")
	s.state = stateProposalCollection
	go s.trackProposalWindow()
}

func (s *synchronizerService) trackProposalWindow() {
	time.Sleep(s.proposalCollectionInterval())
	s.stateMutex.Lock()
	defer s.stateMutex.Unlock()
	logger.Info("Changing state from proposalCollection to evaluation")
	s.state = stateEvaluation
	s.Evaluate()
}

func (s *synchronizerService) registrationInterval() time.Duration {
	const (
		name            = "synchronizer.registrationIntervalMs"
		defaultInterval = 3000 * time.Millisecond
	)

	if !s.cfg.IsSet(name) {
		return defaultInterval
	}

	return s.cfg.GetDuration(name)
}

func (s *synchronizerService) proposalCollectionInterval() time.Duration {
	const (
		name            = "synchronizer.proposalCollectionIntervalMs"
		defaultInterval = 3000 * time.Millisecond
	)

	if !s.cfg.IsSet(name) {
		return defaultInterval
	}

	return s.cfg.GetDuration(name)
}

func getMatchIds(matches []*pb.Match) []string {
	var result []string
	for _, m := range matches {
		result = append(result, m.MatchId)
	}

	return result
}

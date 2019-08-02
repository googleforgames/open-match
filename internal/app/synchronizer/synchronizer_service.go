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

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/ipb"
	"open-match.dev/open-match/internal/statestore"
)

// The service implementing the Synchronizer API that synchronizes the evaluation
// of proposals from Match functions.
type synchronizerService struct {
	state *synchronizerState
}

func newSynchronizerService(cfg config.View, eval evaluator, store statestore.Service) *synchronizerService {
	return &synchronizerService{
		state: newSynchronizerState(cfg, eval, store),
	}
}

// Register associates this request with the current synchronization cycle and
// returns an identifier for this registration. The caller returns this
// identifier back in the evaluation request. This enables synchronizer to
// identify stale evaluation requests belonging to a prior cycle.
func (s *synchronizerService) Register(ctx context.Context, req *ipb.RegisterRequest) (*ipb.RegisterResponse, error) {
	syncState := s.state
	// Registration calls are only permitted if the synchronizer is idle or in
	// registration state. If the synchronizer is in any other state, the
	// registration call blocks on the idle condition. This condition is broadcasted
	// when the previous synchronization cycle is complete, waking up all the
	// blocked requests to make progress in the new cycle.
	syncState.stateMutex.Lock()
	for (syncState.status != statusIdle) && (syncState.status != statusRequestRegistration) {
		syncState.idleCond.Wait()
	}

	defer syncState.stateMutex.Unlock()
	if syncState.status == statusIdle {
		// After waking up, the first request that encounters the idle state changes
		// state to requestRegistration and initializes the state for this synchronization
		// cycle. Consequent registration requests bypass this initialization.
		logger.Info("Changing status from idle to requestRegistration")
		syncState.status = statusRequestRegistration
		syncState.resetCycleData()

		// This method triggers the goroutine that changes the state of the synchronizer
		// to proposalCollection after a configured time.
		go syncState.trackRegistrationWindow()
	}

	// Now that we are in requestRegistration state, allocate an id and initialize
	// request data for this request.
	id := xid.New().String()
	syncState.cycleData.idToRequestData[id] = &requestData{}
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
	syncState := s.state

	logger.WithFields(logrus.Fields{
		"id":        req.GetId(),
		"proposals": getMatchIds(req.GetMatches()),
	}).Info("Received request to evaluate propsals")
	// pendingRequests keeps track of number of requests pending. This is incremented
	// in addProposals while holding the state mutex. The count should be decremented
	// only after this request completes.
	err := syncState.addProposals(req.Id, req.Matches)
	if err != nil {
		return nil, err
	}

	// If proposals were successfully added, then the pending request count has been
	// incremented and should only be reduced when this request is completed.
	defer syncState.cycleData.pendingRequests.Done()

	// fetchResults blocks till results are available. It does not change the
	// state of the synchronizer itself (since multiple concurrent of these can
	// be in progress). The evaluation routine tracks the completion of all of
	// these and then changes the state of the synchronizer to idle.
	results, err := syncState.fetchResults(req.Id)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for _, match := range results {
		for _, ticket := range match.GetTickets() {
			ids = append(ids, ticket.GetId())
		}
	}
	err = syncState.store.AddTicketsToIgnoreList(ctx, ids)
	if err != nil {
		return nil, err
	}

	return &ipb.EvaluateProposalsResponse{Matches: results}, nil
}

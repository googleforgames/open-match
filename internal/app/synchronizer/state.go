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

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/pkg/pb"
)

type synchronizerStatus int

// The Synchronizer can be in one of the following states.
const (
	statusIdle synchronizerStatus = iota
	statusRequestRegistration
	statusProposalCollection
	statusEvaluation
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
type synchronizerState struct {
	cfg        config.View        // Open Match configuration
	cycleData  *synchronizerData  // State for the current synchronization cycle
	eval       evaluator          // Evaluation function to be triggered
	idleCond   *sync.Cond         // Signal any blocked registrations to proceed
	status     synchronizerStatus // Current status of the Synchronizer
	stateMutex sync.Mutex         // Mutex to check on current state and take specific actions.
}

func newSynchronizerState(cfg config.View, eval evaluator) *synchronizerState {
	syncState := &synchronizerState{
		status: statusIdle,
		cfg:    cfg,
		eval:   eval,
		cycleData: &synchronizerData{
			idToRequestData: make(map[string]*requestData),
			resultsReady:    make(chan struct{}),
		},
	}
	syncState.idleCond = sync.NewCond(&syncState.stateMutex)
	return syncState
}

func (syncState *synchronizerState) resetCycleData() {
	syncState.cycleData = &synchronizerData{
		idToRequestData: make(map[string]*requestData),
		resultsReady:    make(chan struct{}),
	}
}

func (syncState *synchronizerState) addProposals(id string, proposals []*pb.Match) error {
	syncState.stateMutex.Lock()
	defer syncState.stateMutex.Unlock()
	if !syncState.canAcceptProposal() {
		return status.Error(codes.DeadlineExceeded, "synchronizer currently not accepting match proposals")
	}

	if _, ok := syncState.cycleData.idToRequestData[id]; !ok {
		return status.Error(codes.DeadlineExceeded, "request not present in the current synchronization cycle")
	}

	if len(syncState.cycleData.idToRequestData[id].proposals) > 0 {
		// We do not currently support submitting multiple separate proposals for a registered request.
		return status.Error(codes.Internal, "multiple proposal addition for a request not supported")
	}

	// Proposal can be added for this request so increment the pending requests
	syncState.cycleData.pendingRequests.Add(1)

	// Copy the proposals being added to the request data for the specified id.
	syncState.cycleData.idToRequestData[id].proposals = make([]*pb.Match, len(proposals))
	copy(syncState.cycleData.idToRequestData[id].proposals, proposals)
	return nil
}

func (syncState *synchronizerState) canAcceptProposal() bool {
	return syncState.status == statusProposalCollection || syncState.status == statusRequestRegistration
}

func (syncState *synchronizerState) fetchResults(id string) ([]*pb.Match, error) {
	t := time.Now()
	var results []*pb.Match

	// fetchMatches will block till evaluation is in progress. The resultsReady channel
	// is closed only when evaluation is complete. This is used to signal all waiting
	// requests that the results are ready.
	// TODO: Add a timeout in case results are not ready within a certain duration.
	<-syncState.cycleData.resultsReady
	if syncState.cycleData.evalError != nil {
		logger.WithError(syncState.cycleData.evalError).Errorf(
			"Evaluation failed for request %v", id)
		return nil, syncState.cycleData.evalError
	}

	results = make([]*pb.Match, len(syncState.cycleData.idToRequestData[id].results))
	copy(results, syncState.cycleData.idToRequestData[id].results)
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
func (syncState *synchronizerState) evaluate(ctx context.Context) {
	aggregateProposals := []*pb.Match{}
	proposalMap := make(map[string]string)
	for id, data := range syncState.cycleData.idToRequestData {
		aggregateProposals = append(aggregateProposals, data.proposals...)
		for _, m := range data.proposals {
			proposalMap[m.GetMatchId()] = id
		}
	}

	logger.WithFields(logrus.Fields{
		"proposals": getMatchIds(aggregateProposals),
	}).Info("Requesting evaluation of proposals")
	results, err := syncState.eval.evaluate(ctx, aggregateProposals)
	if err != nil {
		// Evaluation failed. Set the error on the synchronization cycle data and
		// signal completion of evaluation. Do not process any partial results.
		syncState.cycleData.evalError = err
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

			syncState.cycleData.idToRequestData[cid].results = append(syncState.cycleData.idToRequestData[cid].results, m)
		}
	}

	// Signal completion of evaluation to all requests waiting for results.
	close(syncState.cycleData.resultsReady)

	// Wait for all fetchMatches to complete processing results and then set synchronizer
	// to idle state and signal any blocked registration requests to proceed.
	syncState.cycleData.pendingRequests.Wait()
	logger.Info("Changing state from evaluating to idle")
	syncState.status = statusIdle
	syncState.idleCond.Broadcast()
}

func (syncState *synchronizerState) trackRegistrationWindow() {
	time.Sleep(syncState.registrationInterval())
	syncState.stateMutex.Lock()
	defer syncState.stateMutex.Unlock()
	logger.Infof("Changing state from requestRegistration to proposalCollection after %v", syncState.registrationInterval())
	syncState.status = statusProposalCollection
	go syncState.trackProposalWindow()
}

func (syncState *synchronizerState) trackProposalWindow() {
	time.Sleep(syncState.proposalCollectionInterval())
	syncState.stateMutex.Lock()
	defer syncState.stateMutex.Unlock()
	logger.Infof("Changing status from proposalCollection to evaluation after %v", syncState.proposalCollectionInterval())
	syncState.status = statusEvaluation
	syncState.evaluate(context.Background())
}

func (syncState *synchronizerState) registrationInterval() time.Duration {
	const (
		name            = "synchronizer.registrationIntervalMs"
		defaultInterval = 3000 * time.Millisecond
	)

	if !syncState.cfg.IsSet(name) {
		return defaultInterval
	}

	return syncState.cfg.GetDuration(name)
}

func (syncState *synchronizerState) proposalCollectionInterval() time.Duration {
	const (
		name            = "synchronizer.proposalCollectionIntervalMs"
		defaultInterval = 3000 * time.Millisecond
	)

	if !syncState.cfg.IsSet(name) {
		return defaultInterval
	}

	return syncState.cfg.GetDuration(name)
}

func getMatchIds(matches []*pb.Match) []string {
	var result []string
	for _, m := range matches {
		result = append(result, m.GetMatchId())
	}
	return result
}

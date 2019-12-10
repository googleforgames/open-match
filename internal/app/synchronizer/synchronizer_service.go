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

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/ipb"
	"open-match.dev/open-match/internal/statestore"
	"open-match.dev/open-match/internal/util"
	"open-match.dev/open-match/pkg/pb"
)

// The service implementing the Synchronizer API that synchronizes the evaluation
// of proposals from Match functions.
type synchronizerService struct {
	state *synchronizerState
	store statestore.Service
}

func newSynchronizerService(cfg config.View, eval evaluator, store statestore.Service) *synchronizerService {
	return &synchronizerService{
		state: newSynchronizerState(cfg, eval),
		store: store,
	}
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
	s.state.stateMutex.Lock()
	for (s.state.status != statusIdle) && (s.state.status != statusRequestRegistration) {
		s.state.idleCond.Wait()
	}

	defer s.state.stateMutex.Unlock()
	if s.state.status == statusIdle {
		// After waking up, the first request that encounters the idle state changes
		// state to requestRegistration and initializes the state for this synchronization
		// cycle. Consequent registration requests bypass this initialization.
		logger.Info("Changing status from idle to requestRegistration")
		s.state.status = statusRequestRegistration
		s.state.resetCycleData()

		// This method triggers the goroutine that changes the state of the synchronizer
		// to proposalCollection after a configured time.
		go s.state.trackRegistrationWindow()
	}

	// Now that we are in requestRegistration state, allocate an id and initialize
	// request data for this request.
	id := xid.New().String()
	s.state.cycleData.idToRequestData[id] = &requestData{}
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
func (s *synchronizerService) EvaluateProposals(stream ipb.Synchronizer_EvaluateProposalsServer) error {
	ctx := stream.Context()
	id := util.GetSynchronizerContextID(ctx)
	if len(id) == 0 {
		return status.Errorf(codes.Internal, "synchronizer context id should be an non-empty string")
	}

	var proposals = []*pb.Match{}
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		proposals = append(proposals, req.GetMatch())
	}

	results, err := s.doEvaluateProposals(stream.Context(), proposals, id)
	if err != nil {
		return err
	}

	for _, result := range results {
		if err = stream.Send(&ipb.EvaluateProposalsResponse{Match: result}); err != nil {
			return err
		}
	}

	return nil
}

func (s *synchronizerService) doEvaluateProposals(ctx context.Context, proposals []*pb.Match, id string) ([]*pb.Match, error) {
	// pendingRequests keeps track of number of requests pending. This is incremented
	// in addProposals while holding the state mutex. The count should be decremented
	// only after this request completes.
	err := s.state.addProposals(id, proposals)
	if err != nil {
		return nil, err
	}

	// If proposals were successfully added, then the pending request count has been
	// incremented and should only be reduced when this request is completed.
	defer s.state.cycleData.pendingRequests.Done()

	// fetchResults blocks till results are available. It does not change the
	// state of the synchronizer itself (since multiple concurrent of these can
	// be in progress). The evaluation routine tracks the completion of all of
	// these and then changes the state of the synchronizer to idle.
	results, err := s.state.fetchResults(id)
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

	return results, nil
}

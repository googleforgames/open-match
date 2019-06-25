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
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	ipb "open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/pkg/pb"
)

type testEvaluatorClient struct {
	evalFunc func([]*pb.Match) ([]*pb.Match, error)
}

func (s *testEvaluatorClient) evaluate(proposals []*pb.Match) ([]*pb.Match, error) {
	return s.evalFunc(proposals)
}

type testCallData struct {
	registerDelay      int
	evaluateDelay      int
	proposals          []*pb.Match
	evaluationrErrCode codes.Code
	wantResults        []*pb.Match
}

type testEvaluatorData struct {
	callCount int
	eval      evaluator
	results   [][]*pb.Match
}

type testData struct {
	description   string
	testCalls     []*testCallData
	testEvaluator *testEvaluatorData
	regInterval   string
	propInterval  string
}

func TestSynchronizerService(t *testing.T) {
	assert := assert.New(t)
	// Generate some test matches to be used in the test data.
	tm := []*pb.Match{}
	for i := 0; i < 30; i++ {
		tm = append(tm, &pb.Match{MatchId: xid.New().String()})
	}

	testCases := []*testData{
		{
			description: "basic evaluation scenario, multiple calls with valid results in a registration window",
			testCalls: []*testCallData{
				{
					proposals:   tm[0:5],
					wantResults: tm[0:5],
				},
				{
					proposals:   tm[5:10],
					wantResults: tm[5:10],
				},
			},
			testEvaluator: &testEvaluatorData{
				results: [][]*pb.Match{tm[0:10]},
			},
			regInterval:  "500ms",
			propInterval: "500ms",
		},
		{
			description: "basic evaluation sending subset of proposals in proposal acceptance window",
			testCalls: []*testCallData{
				{
					proposals:     tm[0:5],
					wantResults:   []*pb.Match{tm[0], tm[2]},
					evaluateDelay: 200,
				},
				{
					proposals:     tm[5:10],
					wantResults:   []*pb.Match{tm[5], tm[8], tm[9]},
					evaluateDelay: 1200,
				},
			},
			testEvaluator: &testEvaluatorData{
				results: [][]*pb.Match{{tm[0], tm[2], tm[5], tm[8], tm[9]}},
			},
			regInterval:  "1000ms",
			propInterval: "1000ms",
		},
		{
			description: "Evaluation proposals miss the evaluation window",
			testCalls: []*testCallData{
				{
					proposals:   tm[0:5],
					wantResults: tm[0:5],
				},
				{
					evaluateDelay:      1000,
					proposals:          tm[5:10],
					evaluationrErrCode: codes.DeadlineExceeded,
				},
			},
			testEvaluator: &testEvaluatorData{
				results: [][]*pb.Match{tm[0:5]},
			},
			regInterval:  "200ms",
			propInterval: "200ms",
		},
		{
			description: "Blocking registration requests coming in during proposal acceptance",
			testCalls: []*testCallData{
				{
					proposals:   tm[0:5],
					wantResults: tm[0:5],
				},
				{
					registerDelay: 1000,
					proposals:     tm[5:10],
					wantResults:   tm[5:10],
				},
			},
			testEvaluator: &testEvaluatorData{
				results: [][]*pb.Match{tm[0:5], tm[5:10]},
			},
			regInterval:  "200ms",
			propInterval: "1000ms",
		},
		{
			description: "Mix of successful requests and requests missing evaluation window",
			testCalls: []*testCallData{
				{
					evaluateDelay:      3500,
					proposals:          tm[0:5],
					evaluationrErrCode: codes.DeadlineExceeded,
				},
				{
					evaluateDelay:      3500,
					proposals:          tm[5:10],
					evaluationrErrCode: codes.DeadlineExceeded,
				},
				{
					registerDelay: 3000,
					evaluateDelay: 100,
					proposals:     tm[10:15],
					wantResults:   tm[10:15],
				},
				{
					registerDelay: 3000,
					evaluateDelay: 100,
					proposals:     tm[15:20],
					wantResults:   tm[15:20],
				},
			},
			testEvaluator: &testEvaluatorData{
				results: [][]*pb.Match{[]*pb.Match{}, tm[10:20]},
			},
			regInterval:  "2000ms",
			propInterval: "100ms",
		},
	}

	for _, tc := range testCases {
		tc.testEvaluator.eval = &testEvaluatorClient{
			evalFunc: func(proposals []*pb.Match) ([]*pb.Match, error) {
				if tc.testEvaluator.callCount >= len(tc.testEvaluator.results) {
					assert.Fail("Evaluation triggered more than the expected count")
				}

				result := tc.testEvaluator.results[tc.testEvaluator.callCount]
				tc.testEvaluator.callCount = tc.testEvaluator.callCount + 1
				return result, nil
			},
		}

		runEvaluationTest(t, tc)
	}
}

func runEvaluationTest(t *testing.T, tc *testData) {
	assert := assert.New(t)
	require := require.New(t)
	ctx := context.Background()

	// Generate a config view with paths to the manifests
	cfg := viper.New()
	cfg.Set("synchronizer.proposalCollectionIntervalMs", tc.propInterval)
	cfg.Set("synchronizer.registrationIntervalMs", tc.regInterval)

	s := newSynchronizerService(cfg, tc.testEvaluator.eval)
	require.NotNil(s)
	var w sync.WaitGroup
	w.Add(len(tc.testCalls))
	for _, c := range tc.testCalls {
		c := c
		go func() {
			defer w.Done()
			time.Sleep(time.Duration(c.registerDelay) * time.Millisecond)
			rResp, err := s.Register(ctx, &ipb.RegisterRequest{})
			require.NotNil(rResp)
			require.Nil(err)
			time.Sleep(time.Duration(c.evaluateDelay) * time.Millisecond)
			epResp, err := s.EvaluateProposals(ctx, &ipb.EvaluateProposalsRequest{Match: c.proposals, Id: rResp.Id})
			require.Equal(c.evaluationrErrCode, status.Convert(err).Code())
			if c.evaluationrErrCode == codes.OK {
				require.NotNil(epResp)
				t.Logf(tc.description)
				assert.ElementsMatch(c.wantResults, epResp.Match)
			}
		}()
	}

	w.Wait()
}

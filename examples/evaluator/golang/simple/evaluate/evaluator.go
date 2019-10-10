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

package evaluate

import (
	"math"
	"sort"

	"github.com/golang/protobuf/ptypes"
	"github.com/sirupsen/logrus"
	harness "open-match.dev/open-match/pkg/harness/evaluator/golang"
	"open-match.dev/open-match/pkg/pb"
)

type matchInp struct {
	match *pb.Match
	inp   *pb.DefaultEvaluationCriteria
}

// Evaluate is where your custom evaluation logic lives.
// This sample evaluator sorts and deduplicates the input matches.
func Evaluate(p *harness.EvaluatorParams) ([]*pb.Match, error) {
	matches := make([]*matchInp, 0, len(p.Matches))
	for _, m := range p.Matches {
		// Evaluation criteria is optional, but sort it lower than any matches which
		// provided criteria.
		inp := &pb.DefaultEvaluationCriteria{
			Score: math.Inf(-1),
		}
		if m.EvaluationInput != nil {
			err := ptypes.UnmarshalAny(m.EvaluationInput, inp)
			if err != nil {
				p.Logger.WithFields(logrus.Fields{
					"match_id": m.MatchId,
					"error":    err,
				}).Error("Failed to unmarshal match's DefaultEvaluationCriteria.  Rejecting match.")
				continue
			}
		}
		matches = append(matches, &matchInp{
			match: m,
			inp:   inp,
		})
	}

	sort.Sort(byScore(matches))

	results := []*pb.Match{}
	ticketsUsed := map[string]struct{}{}

loopPotentialMatches:
	for _, m := range matches {
		for _, t := range m.match.GetTickets() {
			if _, ok := ticketsUsed[t.Id]; ok {
				continue loopPotentialMatches
			}
		}

		for _, t := range m.match.GetTickets() {
			ticketsUsed[t.Id] = struct{}{}
		}
		results = append(results, m.match)
	}

	return results, nil
}

type byScore []*matchInp

func (m byScore) Len() int {
	return len(m)
}

func (m byScore) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m byScore) Less(i, j int) bool {
	return m[i].inp.Score > m[j].inp.Score
}

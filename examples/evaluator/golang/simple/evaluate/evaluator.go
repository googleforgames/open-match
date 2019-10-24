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
	nilEvlautionInputs := 0

	for _, m := range p.Matches {
		// Evaluation criteria is optional, but sort it lower than any matches which
		// provided criteria.
		inp := &pb.DefaultEvaluationCriteria{
			Score: math.Inf(-1),
		}

		if a, ok := m.Extensions["evaluation_input"]; ok {
			err := ptypes.UnmarshalAny(a, inp)
			if err != nil {
				p.Logger.WithFields(logrus.Fields{
					"match_id": m.MatchId,
					"error":    err,
				}).Error("Failed to unmarshal match's DefaultEvaluationCriteria.  Rejecting match.")
				continue
			}
		} else {
			nilEvlautionInputs++
		}
		matches = append(matches, &matchInp{
			match: m,
			inp:   inp,
		})
	}

	if nilEvlautionInputs > 0 {
		p.Logger.WithFields(logrus.Fields{
			"count": nilEvlautionInputs,
		}).Info("Some matches don't have the optional field evaluation_input set.")
	}

	sort.Sort(byScore(matches))

	d := decollider{
		ticketsUsed: map[string]struct{}{},
	}

	for _, m := range matches {
		d.maybeAdd(m)
	}

	return d.results, nil
}

type decollider struct {
	results     []*pb.Match
	ticketsUsed map[string]struct{}
}

func (d *decollider) maybeAdd(m *matchInp) {
	for _, t := range m.match.GetTickets() {
		if _, ok := d.ticketsUsed[t.Id]; ok {
			return
		}
	}

	for _, t := range m.match.GetTickets() {
		d.ticketsUsed[t.Id] = struct{}{}
	}

	d.results = append(d.results, m.match)
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

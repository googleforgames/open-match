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
	"open-match.dev/open-match/examples"
	harness "open-match.dev/open-match/pkg/harness/evaluator/golang"
	"open-match.dev/open-match/pkg/pb"
)

// Evaluate is where your custom evaluation logic lives.
// This sample evaluator sorts and deduplicates the input matches.
func Evaluate(p *harness.EvaluatorParams) ([]*pb.Match, error) {
	scoreInDescendingOrder := func(a, b *pb.Match) bool {
		return a.GetProperties().GetFields()[examples.MatchScore].GetNumberValue() > b.GetProperties().GetFields()[examples.MatchScore].GetNumberValue()
	}
	by(scoreInDescendingOrder).Sort(p.Matches)

	results := []*pb.Match{}
	dedup := map[string]bool{}

	for _, match := range p.Matches {
		if isNonCollidingMatch(match, dedup) {
			for _, ticket := range match.GetTickets() {
				dedup[ticket.GetId()] = true
			}
			results = append(results, match)
		}
	}

	return results, nil
}

func isNonCollidingMatch(match *pb.Match, validTickets map[string]bool) bool {
	for _, ticket := range match.GetTickets() {
		id := ticket.GetId()
		if _, ok := validTickets[id]; ok {
			return false
		}
	}
	return true
}

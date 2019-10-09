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

// Evaluate is where your custom evaluation logic lives.
// This sample evaluator sorts and deduplicates the input matches.
func Evaluate(p *harness.EvaluatorParams) ([]*pb.Match, error) {
	matches := make(matcheExts, 0, len(p.Matches))
	for _, match := range p.Matches {
		// Evaluation criteria is optional, but sort it lower than any matches which
		// provided criteria.
		ext := &pb.DefaultEvaluationCriteria{
			Score: math.Inf(-1),
		}
		if match.Extension != nil {
			err := ptypes.UnmarshalAny(match.Extension, ext)
			if err != nil {
				p.Logger.WithFields(logrus.Fields{
					"match_id": match.MatchId,
					"error":    err,
				}).Error("Failed to unmarshal match's DefaultEvaluationCriteria.  Rejecting match.")
				continue
			}
		}
		matches = append(matches, &matchExt{
			match: match,
			ext:   ext,
		})
	}

	sort.Sort(matches)

	results := []*pb.Match{}
	ticketsUsed := map[string]struct{}{}

outer:
	for _, matchExt := range matches {
		for _, ticket := range matchExt.match.GetTickets() {
			if _, ok := ticketsUsed[ticket.Id]; ok {
				continue outer
			}
		}

		for _, ticket := range matchExt.match.GetTickets() {
			ticketsUsed[ticket.Id] = struct{}{}
		}
		results = append(results, matchExt.match)
	}

	return results, nil
}

type matchExt struct {
	match *pb.Match
	ext   *pb.DefaultEvaluationCriteria
}

type matcheExts []*matchExt

func (matches matcheExts) Len() int {
	return len(matches)
}

func (matches matcheExts) Swap(i, j int) {
	matches[i], matches[j] = matches[j], matches[i]
}

func (matches matcheExts) Less(i, j int) bool {
	return matches[i].ext.Score > matches[j].ext.Score
}

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

// Package defaulteval provides a simple score based evaluator.
package defaulteval

import (
	"context"
	"math"
	"sort"

	"go.opencensus.io/stats"

	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats/view"
	"open-match.dev/open-match/internal/app/evaluator"
	"open-match.dev/open-match/internal/appmain"
	"open-match.dev/open-match/pkg/pb"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "evaluator",
		"component": "evaluator.default",
	})

	collidedMatchesPerEvaluate     = stats.Int64("open-match.dev/defaulteval/collided_matches_per_call", "Number of collided matches per default evaluator call", stats.UnitDimensionless)
	collidedMatchesPerEvaluateView = &view.View{
		Measure:     collidedMatchesPerEvaluate,
		Name:        "open-match.dev/defaulteval/collided_matches_per_call",
		Description: "Number of collided matches per default evaluator call",
		Aggregation: view.Sum(),
	}
)

type matchInp struct {
	match *pb.Match
	inp   *pb.DefaultEvaluationCriteria
}

// BindService define the initialization steps for this evaluator
func BindService(p *appmain.Params, b *appmain.Bindings) error {
	if err := evaluator.BindServiceFor(evaluate)(p, b); err != nil {
		return err
	}
	b.RegisterViews(collidedMatchesPerEvaluateView)
	return nil
}

// evaluate sorts the matches by DefaultEvaluationCriteria.Score (optional),
// then returns matches which don't collide with previously returned matches.
func evaluate(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
	matches := make([]*matchInp, 0)
	nilEvaluationInputs := 0

	for m := range in {
		// Evaluation criteria is optional, but sort it lower than any matches which
		// provided criteria.
		inp := &pb.DefaultEvaluationCriteria{
			Score: math.Inf(-1),
		}

		if a, ok := m.Extensions["evaluation_input"]; ok {
			err := a.UnmarshalTo(inp)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"match_id": m.MatchId,
					"error":    err,
				}).Error("Failed to unmarshal match's DefaultEvaluationCriteria.  Rejecting match.")
				continue
			}
		} else {
			nilEvaluationInputs++
		}
		matches = append(matches, &matchInp{
			match: m,
			inp:   inp,
		})
	}

	if nilEvaluationInputs > 0 {
		logger.WithFields(logrus.Fields{
			"count": nilEvaluationInputs,
		}).Info("Some matches don't have the optional field evaluation_input set.")
	}

	sort.Sort(byScore(matches))

	d := decollider{
		ticketsUsed:   make(map[string]*collidingMatch),
		backfillsUsed: make(map[string]*collidingMatch),
	}

	for _, m := range matches {
		d.maybeAdd(m)
	}

	stats.Record(context.Background(), collidedMatchesPerEvaluate.M(int64(len(matches)-len(d.resultIDs))))

	for _, id := range d.resultIDs {
		out <- id
	}

	return nil
}

type collidingMatch struct {
	id    string
	score float64
}

type decollider struct {
	resultIDs     []string
	ticketsUsed   map[string]*collidingMatch
	backfillsUsed map[string]*collidingMatch
}

func (d *decollider) maybeAdd(m *matchInp) {
	if m.match.Backfill != nil && m.match.Backfill.Id != "" {
		if cm, ok := d.backfillsUsed[m.match.Backfill.Id]; ok {
			logger.WithFields(logrus.Fields{
				"match_id":              m.match.GetMatchId(),
				"backfill_id":           m.match.Backfill.Id,
				"match_score":           m.inp.GetScore(),
				"colliding_match_id":    cm.id,
				"colliding_match_score": cm.score,
			}).Info("Higher quality match with colliding backfill found. Rejecting match.")
			return
		}
	}

	for _, t := range m.match.GetTickets() {
		if cm, ok := d.ticketsUsed[t.Id]; ok {
			logger.WithFields(logrus.Fields{
				"match_id":              m.match.GetMatchId(),
				"ticket_id":             t.GetId(),
				"match_score":           m.inp.GetScore(),
				"colliding_match_id":    cm.id,
				"colliding_match_score": cm.score,
			}).Info("Higher quality match with colliding ticket found. Rejecting match.")
			return
		}
	}

	if m.match.Backfill != nil && m.match.Backfill.Id != "" {
		d.backfillsUsed[m.match.Backfill.Id] = &collidingMatch{
			id:    m.match.GetMatchId(),
			score: m.inp.GetScore(),
		}
	}

	for _, t := range m.match.GetTickets() {
		d.ticketsUsed[t.Id] = &collidingMatch{
			id:    m.match.GetMatchId(),
			score: m.inp.GetScore(),
		}
	}

	d.resultIDs = append(d.resultIDs, m.match.GetMatchId())
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

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
	"log"
	"sort"
	"time"

	"open-match.dev/open-match/pkg/pb"
)

type matchEval struct {
	match   *pb.Match
	maxWait float64
}

// Evaluate is where your custom evaluation logic lives.
// This sample evaluator sorts and deduplicates the input matches.
func evaluate(proposals []*pb.Match) ([]string, error) {
	matches := make([]*matchEval, 0, len(proposals))
	now := float64(time.Now().UnixNano())
	for _, m := range proposals {
		var maxWait float64
		for _, t := range m.GetTickets() {
			qt := now - t.GetSearchFields().GetDoubleArgs()["time.enterqueue"]
			if qt > maxWait {
				maxWait = qt
			}
		}

		matches = append(matches, &matchEval{
			match:   m,
			maxWait: maxWait,
		})
	}

	sort.Sort(byMaxWait(matches))

	d := decollider{
		ticketsUsed: make(map[string]*collidingMatch),
	}

	for _, m := range matches {
		d.maybeAdd(m)
	}

	return d.resultIDs, nil
}

type collidingMatch struct {
	id      string
	maxWait float64
}

type decollider struct {
	resultIDs   []string
	ticketsUsed map[string]*collidingMatch
}

func (d *decollider) maybeAdd(m *matchEval) {
	for _, t := range m.match.GetTickets() {
		if cm, ok := d.ticketsUsed[t.Id]; ok {
			log.Printf("Rejecting Match %v(max_wait:%v) as its ticket %v collides with Match %v (max_wait:%v).",
				m.match.GetMatchId(), m.maxWait, t.GetId(), cm.id, cm.maxWait)
			return
		}
	}

	for _, t := range m.match.GetTickets() {
		d.ticketsUsed[t.Id] = &collidingMatch{
			id:      m.match.GetMatchId(),
			maxWait: m.maxWait,
		}
	}

	d.resultIDs = append(d.resultIDs, m.match.GetMatchId())
}

type byMaxWait []*matchEval

func (m byMaxWait) Len() int {
	return len(m)
}

func (m byMaxWait) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m byMaxWait) Less(i, j int) bool {
	return m[i].maxWait > m[j].maxWait
}

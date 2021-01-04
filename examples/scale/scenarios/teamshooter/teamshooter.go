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

// TeamShooterScenario is a scenario which is designed to emulate the
// approximate behavior to open match that a skill based team game would have.
// It doesn't try to provide good matchmaking for real players. There are three
// arguments used:
// mode: The game mode the players wants to play in. mode is a hard partition.
// regions: Players may have good latency to one or more regions. A player will
//   search for matches in all eligible regions.
// skill: Players have a random skill based on a normal distribution. Players
//   will only be matched with other players who have a close skill value. The
//   match functions have overlapping partitions of the skill brackets.
package teamshooter

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/pkg/pb"
)

const (
	poolName = "all"
	skillArg = "skill"
	modeArg  = "mode"
)

// TeamShooterScenario provides the required methods for running a scenario.
type TeamShooterScenario struct {
	// Names of available region tags.
	regions []string
	// Maximum regions a player can search in.
	maxRegions int
	// Number of tickets which form a match.
	playersPerGame int
	// For each pair of consequitive values, the value to split profiles on by
	// skill.
	skillBoundaries []float64
	// Maximum difference between two tickets to consider a match valid.
	maxSkillDifference float64
	// List of mode names.
	modes []string
	// Returns a random mode, with some weight.
	randomMode func() string
}

// Scenario creates a new TeamShooterScenario.
func Scenario() *TeamShooterScenario {

	modes, randomMode := weightedChoice(map[string]int{
		"pl": 100, // Payload, very popular.
		"cp": 25,  // Capture point, 1/4 as popular.
	})

	regions := []string{}
	for i := 0; i < 2; i++ {
		regions = append(regions, fmt.Sprintf("region_%d", i))
	}

	return &TeamShooterScenario{
		regions:            regions,
		maxRegions:         1,
		playersPerGame:     12,
		skillBoundaries:    []float64{math.Inf(-1), 0, math.Inf(1)},
		maxSkillDifference: 0.01,
		modes:              modes,
		randomMode:         randomMode,
	}
}

// Profiles shards the player base on mode, region, and skill.
func (t *TeamShooterScenario) Profiles() []*pb.MatchProfile {
	p := []*pb.MatchProfile{}

	for _, region := range t.regions {
		for _, mode := range t.modes {
			for i := 0; i+1 < len(t.skillBoundaries); i++ {
				skillMin := t.skillBoundaries[i] - t.maxSkillDifference/2
				skillMax := t.skillBoundaries[i+1] + t.maxSkillDifference/2
				p = append(p, &pb.MatchProfile{
					Name: fmt.Sprintf("%s_%s_%v-%v", region, mode, skillMin, skillMax),
					Pools: []*pb.Pool{
						{
							Name: poolName,
							DoubleRangeFilters: []*pb.DoubleRangeFilter{
								{
									DoubleArg: skillArg,
									Min:       skillMin,
									Max:       skillMax,
								},
							},
							TagPresentFilters: []*pb.TagPresentFilter{
								{
									Tag: region,
								},
							},
							StringEqualsFilters: []*pb.StringEqualsFilter{
								{
									StringArg: modeArg,
									Value:     mode,
								},
							},
						},
					},
				})
			}
		}
	}

	return p
}

// Ticket creates a randomized player.
func (t *TeamShooterScenario) Ticket() *pb.Ticket {
	region := rand.Intn(len(t.regions))
	numRegions := rand.Intn(t.maxRegions) + 1

	tags := []string{}
	for i := 0; i < numRegions; i++ {
		tags = append(tags, t.regions[region])
		// The Earth is actually a circle.
		region = (region + 1) % len(t.regions)
	}

	return &pb.Ticket{
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				skillArg: clamp(rand.NormFloat64(), -3, 3),
			},
			StringArgs: map[string]string{
				modeArg: t.randomMode(),
			},
			Tags: tags,
		},
	}
}

func (t *TeamShooterScenario) Backfill() *pb.Backfill {
	return nil
}

// MatchFunction puts tickets into matches based on their skill, finding the
// required number of tickets for a game within the maximum skill difference.
func (t *TeamShooterScenario) MatchFunction(p *pb.MatchProfile, poolBackfills map[string][]*pb.Backfill, poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error) {
	skill := func(t *pb.Ticket) float64 {
		return t.SearchFields.DoubleArgs[skillArg]
	}

	tickets := poolTickets[poolName]
	var matches []*pb.Match

	sort.Slice(tickets, func(i, j int) bool {
		return skill(tickets[i]) < skill(tickets[j])
	})

	for i := 0; i+t.playersPerGame <= len(tickets); i++ {
		mt := tickets[i : i+t.playersPerGame]
		if skill(mt[len(mt)-1])-skill(mt[0]) < t.maxSkillDifference {
			avg := float64(0)
			for _, t := range mt {
				avg += skill(t)
			}
			avg /= float64(len(mt))

			q := float64(0)
			for _, t := range mt {
				diff := skill(t) - avg
				q -= diff * diff
			}

			m, err := (&matchExt{
				id:            fmt.Sprintf("profile-%v-time-%v-%v", p.GetName(), time.Now().Format("2006-01-02T15:04:05.00"), len(matches)),
				matchProfile:  p.GetName(),
				matchFunction: "skillmatcher",
				tickets:       mt,
				quality:       q,
			}).pack()
			if err != nil {
				return nil, err
			}
			matches = append(matches, m)
		}
	}

	return matches, nil
}

// Evaluate returns matches in order of highest quality, skipping any matches
// which contain tickets that are already used.
func (t *TeamShooterScenario) Evaluate(stream pb.Evaluator_EvaluateServer) error {
	// Unpacked proposal matches.
	proposals := []*matchExt{}
	// Ticket ids which are used in a match.
	used := map[string]struct{}{}

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Error reading evaluator input stream: %w", err)
		}

		p, err := unpackMatch(req.GetMatch())
		if err != nil {
			return err
		}
		proposals = append(proposals, p)
	}

	// Higher quality is better.
	sort.Slice(proposals, func(i, j int) bool {
		return proposals[i].quality > proposals[j].quality
	})

outer:
	for _, p := range proposals {
		for _, t := range p.tickets {
			if _, ok := used[t.Id]; ok {
				continue outer
			}
		}

		for _, t := range p.tickets {
			used[t.Id] = struct{}{}
		}

		err := stream.Send(&pb.EvaluateResponse{MatchId: p.id})
		if err != nil {
			return fmt.Errorf("Error sending evaluator output stream: %w", err)
		}
	}

	return nil
}

func (b *TeamShooterScenario) Backend() func(pb.BackendServiceClient, pb.FrontendServiceClient, *logrus.Entry) error {
	return nil
}

func (b *TeamShooterScenario) Frontend() func(pb.FrontendServiceClient, *logrus.Entry) error {
	return nil
}

// matchExt presents the match and extension data in a native form, and allows
// easy conversion to and from proto format.
type matchExt struct {
	id            string
	tickets       []*pb.Ticket
	quality       float64
	matchProfile  string
	matchFunction string
}

func unpackMatch(m *pb.Match) (*matchExt, error) {
	v := &wrappers.DoubleValue{}

	err := ptypes.UnmarshalAny(m.Extensions["quality"], v)
	if err != nil {
		return nil, fmt.Errorf("Error unpacking match quality: %w", err)
	}

	return &matchExt{
		id:            m.MatchId,
		tickets:       m.Tickets,
		quality:       v.Value,
		matchProfile:  m.MatchProfile,
		matchFunction: m.MatchFunction,
	}, nil
}

func (m *matchExt) pack() (*pb.Match, error) {
	v := &wrappers.DoubleValue{Value: m.quality}

	a, err := ptypes.MarshalAny(v)
	if err != nil {
		return nil, fmt.Errorf("Error packing match quality: %w", err)
	}

	return &pb.Match{
		MatchId:       m.id,
		Tickets:       m.tickets,
		MatchProfile:  m.matchProfile,
		MatchFunction: m.matchFunction,
		Extensions: map[string]*any.Any{
			"quality": a,
		},
	}, nil
}

func clamp(v float64, min float64, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// weightedChoice takes a map of values, and their relative probability.  It
// returns a list of the values, along with a function which will return random
// choices from the values with the weighted probability.
func weightedChoice(m map[string]int) ([]string, func() string) {
	s := make([]string, 0, len(m))
	total := 0
	for k, v := range m {
		s = append(s, k)
		total += v
	}

	return s, func() string {
		remainder := rand.Intn(total)
		for k, v := range m {
			remainder -= v
			if remainder < 0 {
				return k
			}
		}
		panic("weightedChoice is broken.")
	}
}

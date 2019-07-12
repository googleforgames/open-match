// +build !e2ecluster

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

package e2e

import (
	"fmt"
	"io"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/pkg/structs"
)

type testProfile struct {
	name  string
	pools []*pb.Pool
}

type testTicket struct {
	skill      float64
	mapValue   string
	targetPool string
	id         string
}

func TestMinimatch(t *testing.T) {
	// TODO: Currently, the E2E test uses globally defined test data. Consider
	// improving this in future iterations to test data scoped to sepcific test cases

	// Tickets used for testing. The id will be populated once the Tickets are created.
	sourceTickets := []testTicket{
		{skill: 1, mapValue: e2e.Map1Attribute, targetPool: e2e.Map1BeginnerPool},
		{skill: 5, mapValue: e2e.Map1Attribute, targetPool: e2e.Map1BeginnerPool},
		{skill: 8, mapValue: e2e.Map1Attribute, targetPool: e2e.Map1AdvancedPool},
		{skill: 0, mapValue: e2e.Map2Attribute, targetPool: e2e.Map2BeginnerPool},
		{skill: 6, mapValue: e2e.Map2Attribute, targetPool: e2e.Map2AdvancedPool},
		{skill: 10, mapValue: e2e.Map2Attribute, targetPool: e2e.Map2AdvancedPool},
		{skill: 10, mapValue: "non-indexable-map"},
	}

	// The Pools that are used for testing. Currently, each pool simply represents
	// a combination of the attributes indexed on a ticket.
	testPools := map[string]*pb.Pool{
		e2e.Map1BeginnerPool: {
			Name: e2e.Map1BeginnerPool,
			Filters: []*pb.Filter{
				{Attribute: e2e.SkillAttribute, Min: 0, Max: 5},
				{Attribute: e2e.Map1Attribute, Max: math.MaxFloat64},
			},
		},
		e2e.Map1AdvancedPool: {
			Name: e2e.Map1AdvancedPool,
			Filters: []*pb.Filter{
				{Attribute: e2e.SkillAttribute, Min: 6, Max: 10},
				{Attribute: e2e.Map1Attribute, Max: math.MaxFloat64},
			},
		},
		e2e.Map2BeginnerPool: {
			Name: e2e.Map2BeginnerPool,
			Filters: []*pb.Filter{
				{Attribute: e2e.SkillAttribute, Min: 0, Max: 5},
				{Attribute: e2e.Map2Attribute, Max: math.MaxFloat64},
			},
		},
		e2e.Map2AdvancedPool: {
			Name: e2e.Map2AdvancedPool,
			Filters: []*pb.Filter{
				{Attribute: e2e.SkillAttribute, Min: 6, Max: 10},
				{Attribute: e2e.Map2Attribute, Max: math.MaxFloat64},
			},
		},
	}

	// Test profiles being tested for. Note that each profile embeds two pools - and
	// the current MMF returns a match per pool in the profile - so each profile should
	// output two matches that are comprised of tickets belonging to that pool.
	sourceProfiles := []testProfile{
		{name: "", pools: []*pb.Pool{testPools[e2e.Map1BeginnerPool], testPools[e2e.Map1AdvancedPool]}},
		{name: "", pools: []*pb.Pool{testPools[e2e.Map2BeginnerPool], testPools[e2e.Map2AdvancedPool]}},
	}

	tests := []struct {
		fcGen       func(e2e.OM) *pb.FunctionConfig
		description string
	}{
		{
			func(om e2e.OM) *pb.FunctionConfig { return om.MustMmfConfigGRPC() },
			"grpc config",
		},
		{
			func(om e2e.OM) *pb.FunctionConfig { return om.MustMmfConfigHTTP() },
			"http config",
		},
	}

	t.Run("TestMinimatch", func(t *testing.T) {
		for _, test := range tests {
			test := test
			testTickets := make([]testTicket, len(sourceTickets))
			testProfiles := make([]testProfile, len(sourceProfiles))
			copy(testTickets, sourceTickets)
			copy(testProfiles, sourceProfiles)

			t.Run(fmt.Sprintf("TestMinimatch-%v", test.description), func(t *testing.T) {
				t.Parallel()

				om, closer := e2e.New(t)
				defer closer()
				fc := test.fcGen(om)

				fe := om.MustFrontendGRPC()
				mml := om.MustMmLogicGRPC()

				// Create all the tickets and validate ticket creation succeeds. Also populate ticket ids
				// to expected player pools.
				for i, td := range testTickets {
					resp, err := fe.CreateTicket(om.Context(), &pb.CreateTicketRequest{Ticket: &pb.Ticket{
						Properties: structs.Struct{
							e2e.SkillAttribute: structs.Number(td.skill),
							td.mapValue:        structs.Number(float64(time.Now().Unix())),
						}.S(),
					}})

					assert.NotNil(t, resp)
					assert.Nil(t, err)
					testTickets[i].id = resp.Ticket.Id
				}

				// poolTickets represents a map of the pool name to all the ticket ids in the pool.
				poolTickets := make(map[string][]string)

				// Query tickets for each pool
				for _, pool := range testPools {
					qtstr, err := mml.QueryTickets(om.Context(), &pb.QueryTicketsRequest{Pool: pool})
					assert.Nil(t, err)
					assert.NotNil(t, qtstr)

					var tickets []*pb.Ticket
					for {
						qtresp, err := qtstr.Recv()
						if err == io.EOF {
							break
						}
						assert.Nil(t, err)
						assert.NotNil(t, qtresp)
						tickets = append(tickets, qtresp.Tickets...)
					}

					// Generate a map of pool name to all tickets belonging to the pool.
					for _, ticket := range tickets {
						poolTickets[pool.Name] = append(poolTickets[pool.Name], ticket.Id)
					}

					// Generate a map of pool name to all tickets that should belong to the pool
					// based on test ticket data.
					var want []string
					for _, td := range testTickets {
						if td.targetPool == pool.Name {
							want = append(want, td.id)
						}
					}

					// Validate that all the pools have the expected tickets.
					assert.Equal(t, poolTickets[pool.Name], want)
				}

				testFetchMatches(t, poolTickets, testProfiles, om, fc)
			})
		}
	})
}

func testFetchMatches(t *testing.T, poolTickets map[string][]string, testProfiles []testProfile, om e2e.OM, fc *pb.FunctionConfig) {
	// Fetch Matches for each test profile.
	be := om.MustBackendGRPC()
	for _, profile := range testProfiles {
		br, err := be.FetchMatches(om.Context(), &pb.FetchMatchesRequest{
			Config:   fc,
			Profiles: []*pb.MatchProfile{{Name: profile.name, Pools: profile.pools}},
		})
		assert.Nil(t, err)
		assert.NotNil(t, br)
		assert.NotNil(t, br.GetMatches())

		for _, match := range br.GetMatches() {
			// Currently, the MMF simply creates a match per pool in the match profile - and populates
			// the roster with the pool name. Thus validate that for the roster populated in the match
			// result has all the tickets expected in that pool.
			assert.Equal(t, len(match.GetRosters()), 1)
			assert.Equal(t, match.GetRosters()[0].GetTicketIds(), poolTickets[match.GetRosters()[0].GetName()])

			var gotTickets []string
			for _, ticket := range match.GetTickets() {
				gotTickets = append(gotTickets, ticket.Id)
			}

			// Given that currently we only populate all tickets in a match in a Roster, validate that
			// all the tickets present in the result match are equal to the tickets in the pool for that match.
			assert.Equal(t, gotTickets, poolTickets[match.GetRosters()[0].GetName()])
		}

		// Verify calling fetch matches twice within ttl interval won't yield new results
		br, err = be.FetchMatches(om.Context(), &pb.FetchMatchesRequest{
			Config:   fc,
			Profiles: []*pb.MatchProfile{{Name: profile.name, Pools: profile.pools}},
		})

		assert.Nil(t, br.GetMatches())
		assert.Nil(t, err)
	}
}

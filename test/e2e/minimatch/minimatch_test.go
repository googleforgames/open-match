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

package minimatch

import (
	"io"
	"math"
	"testing"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/stretchr/testify/require"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	"open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
)

type testProfile struct {
	name  string
	pools []*pb.Pool
}

func TestMinimatch(t *testing.T) {
	assert := require.New(t)
	evalTc := createEvaluatorForTest(t)
	defer evalTc.Close()
	minimatchTc := createMinimatchForTest(t, evalTc)
	defer minimatchTc.Close()
	mmfTc := createMatchFunctionForTest(t, minimatchTc)
	defer mmfTc.Close()

	minimatchConn := minimatchTc.MustGRPC()
	fe := pb.NewFrontendClient(minimatchConn)
	mml := pb.NewMmLogicClient(minimatchConn)
	be := pb.NewBackendClient(minimatchConn)

	// TODO: Currently, the E2E test uses globally defined test data. Consider
	// improving this in future iterations to test data scoped to sepcific test cases

	// Tickets used for testing. The id will be populated once the Tickets are created.
	testTickets := []struct {
		skill      float64
		mapValue   string
		targetPool string
		id         string
	}{
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
			Filter: []*pb.Filter{
				{Attribute: e2e.SkillAttribute, Min: 0, Max: 5},
				{Attribute: e2e.Map1Attribute, Max: math.MaxFloat64},
			},
		},
		e2e.Map1AdvancedPool: {
			Name: e2e.Map1AdvancedPool,
			Filter: []*pb.Filter{
				{Attribute: e2e.SkillAttribute, Min: 6, Max: 10},
				{Attribute: e2e.Map1Attribute, Max: math.MaxFloat64},
			},
		},
		e2e.Map2BeginnerPool: {
			Name: e2e.Map2BeginnerPool,
			Filter: []*pb.Filter{
				{Attribute: e2e.SkillAttribute, Min: 0, Max: 5},
				{Attribute: e2e.Map2Attribute, Max: math.MaxFloat64},
			},
		},
		e2e.Map2AdvancedPool: {
			Name: e2e.Map2AdvancedPool,
			Filter: []*pb.Filter{
				{Attribute: e2e.SkillAttribute, Min: 6, Max: 10},
				{Attribute: e2e.Map2Attribute, Max: math.MaxFloat64},
			},
		},
	}

	// Test profiles being tested for. Note that each profile embeds two pools - and
	// the current MMF returns a match per pool in the profile - so each profile should
	// output two matches that are comprised of tickets belonging to that pool.
	testProfiles := []testProfile{
		{name: "", pools: []*pb.Pool{testPools[e2e.Map1BeginnerPool], testPools[e2e.Map1AdvancedPool]}},
		{name: "", pools: []*pb.Pool{testPools[e2e.Map2BeginnerPool], testPools[e2e.Map2AdvancedPool]}},
	}

	// Create all the tickets and validate ticket creation succeeds. Also populate ticket ids
	// to expected player pools.
	for i, td := range testTickets {
		resp, err := fe.CreateTicket(minimatchTc.Context(), &pb.CreateTicketRequest{Ticket: &pb.Ticket{
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					e2e.SkillAttribute: {Kind: &structpb.Value_NumberValue{NumberValue: td.skill}},
					td.mapValue:        {Kind: &structpb.Value_NumberValue{NumberValue: float64(time.Now().Unix())}},
				},
			},
		}})

		assert.NotNil(resp)
		assert.Nil(err)
		testTickets[i].id = resp.Ticket.Id
	}

	// poolTickets represents a map of the pool name to all the ticket ids in the pool.
	poolTickets := make(map[string][]string)

	// Query tickets for each pool
	for _, pool := range testPools {
		qtstr, err := mml.QueryTickets(minimatchTc.Context(), &pb.QueryTicketsRequest{Pool: pool})
		assert.Nil(err)
		assert.NotNil(qtstr)

		var tickets []*pb.Ticket
		for {
			qtresp, err := qtstr.Recv()
			if err == io.EOF {
				break
			}
			assert.Nil(err)
			assert.NotNil(qtresp)
			tickets = append(tickets, qtresp.Ticket...)
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
		assert.Equal(poolTickets[pool.Name], want)
	}

	fcs := []*pb.FunctionConfig{
		{
			Type: pb.FunctionConfig_GRPC,
			Host: mmfTc.GetHostname(),
			Port: int32(mmfTc.GetGRPCPort()),
		},
		{
			Type: pb.FunctionConfig_REST,
			Host: mmfTc.GetHostname(),
			Port: int32(mmfTc.GetHTTPPort()),
		},
	}

	for _, fc := range fcs {
		testFetchMatches(assert, poolTickets, testProfiles, minimatchTc, be, fc)
	}
}

func testFetchMatches(assert *require.Assertions, poolTickets map[string][]string, testProfiles []testProfile, tc *rpcTesting.TestContext, be pb.BackendClient, fc *pb.FunctionConfig) {
	// Fetch Matches for each test profile.
	for _, profile := range testProfiles {
		br, err := be.FetchMatches(tc.Context(), &pb.FetchMatchesRequest{
			Config:  fc,
			Profile: []*pb.MatchProfile{{Name: profile.name, Pool: profile.pools}},
		})

		assert.Nil(err)
		assert.NotNil(br)
		assert.NotNil(br.Match)

		for _, match := range br.GetMatch() {
			// Currently, the MMF simply creates a match per pool in the match profile - and populates
			// the roster with the pool name. Thus validate that for the roster populated in the match
			// result has all the tickets expected in that pool.
			assert.Equal(len(match.GetRoster()), 1)
			assert.Equal(match.GetRoster()[0].TicketId, poolTickets[match.GetRoster()[0].GetName()])

			var gotTickets []string
			for _, ticket := range match.GetTicket() {
				gotTickets = append(gotTickets, ticket.Id)
			}

			// Given that currently we only populate all tickets in a match in a Roster, validate that
			// all the tickets present in the result match are equal to the tickets in the pool for that match.
			assert.Equal(gotTickets, poolTickets[match.GetRoster()[0].GetName()])
		}
	}
}

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
	"github.com/stretchr/testify/assert"
	"open-match.dev/open-match/internal/pb"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
)

const (
	// Names of the test pools used by this test.
	map1BeginnerPool = "map1beginner"
	map1AdvancedPool = "map1advanced"
	map2BeginnerPool = "map2beginner"
	map2AdvancedPool = "map2advanced"
	// Test specific metadata
	skillattribute = "skill"
	map1attribute  = "map1"
	map2attribute  = "map2"
)

type testProfile struct {
	name  string
	pools []*pb.Pool
}

func TestMinimatchStartup(t *testing.T) {
	assert := assert.New(t)
	minimatchTc := createMinimatchForTest(t)
	defer minimatchTc.Close()
	mmfTc, mmfName := createMatchFunctionForTest(t, minimatchTc)
	defer mmfTc.Close()

	// TODO: Currently, the E2E test uses globally defined test data. Consider
	// improving this in future iterations to test data scoped to sepcific test cases

	// Tickets used for testing. The id will be populated once the Tickets are created.
	testTickets := []struct {
		skill      float64
		mapValue   string
		targetPool string
		id         string
	}{
		{skill: 1, mapValue: map1attribute, targetPool: map1BeginnerPool},
		{skill: 5, mapValue: map1attribute, targetPool: map1BeginnerPool},
		{skill: 8, mapValue: map1attribute, targetPool: map1AdvancedPool},
		{skill: 0, mapValue: map2attribute, targetPool: map2BeginnerPool},
		{skill: 6, mapValue: map2attribute, targetPool: map2AdvancedPool},
		{skill: 10, mapValue: map2attribute, targetPool: map2AdvancedPool},
		{skill: 10, mapValue: "non-indexable-map"},
	}

	// The Pools that are used for testing. Currently, each pool simply represents
	// a combination of the attributes indexed on a ticket.
	testPools := map[string]*pb.Pool{
		map1BeginnerPool: {
			Name: map1BeginnerPool,
			Filter: []*pb.Filter{
				{Attribute: skillattribute, Min: 0, Max: 5},
				{Attribute: map1attribute, Max: math.MaxFloat64},
			},
		},
		map1AdvancedPool: {
			Name: map1AdvancedPool,
			Filter: []*pb.Filter{
				{Attribute: skillattribute, Min: 6, Max: 10},
				{Attribute: map1attribute, Max: math.MaxFloat64},
			},
		},
		map2BeginnerPool: {
			Name: map2BeginnerPool,
			Filter: []*pb.Filter{
				{Attribute: skillattribute, Min: 0, Max: 5},
				{Attribute: map2attribute, Max: math.MaxFloat64},
			},
		},
		map2AdvancedPool: {
			Name: map2AdvancedPool,
			Filter: []*pb.Filter{
				{Attribute: skillattribute, Min: 6, Max: 10},
				{Attribute: map2attribute, Max: math.MaxFloat64},
			},
		},
	}

	// Test profiles being tested for. Note that each profile embeds two pools - and
	// the current MMF returns a match per pool in the profile - so each profile should
	// output two matches that are comprised of tickets belonging to that pool.
	testProfiles := []testProfile{
		{name: "", pools: []*pb.Pool{testPools[map1BeginnerPool], testPools[map1AdvancedPool]}},
		{name: "", pools: []*pb.Pool{testPools[map2BeginnerPool], testPools[map2AdvancedPool]}},
	}

	fe := pb.NewFrontendClient(minimatchTc.MustGRPC())

	// Create all the tickets and validate ticket creation succeeds. Also populate ticket ids
	// to expected player pools.
	for i, td := range testTickets {
		resp, err := fe.CreateTicket(minimatchTc.Context(), &pb.CreateTicketRequest{Ticket: &pb.Ticket{
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					skillattribute: {Kind: &structpb.Value_NumberValue{NumberValue: td.skill}},
					td.mapValue:    {Kind: &structpb.Value_NumberValue{NumberValue: float64(time.Now().Unix())}},
				},
			},
		}})

		assert.NotNil(resp)
		assert.Nil(err)
		testTickets[i].id = resp.Ticket.Id
	}

	mml := pb.NewMmLogicClient(minimatchTc.MustGRPC())

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
			Name: mmfName,
			Type: &pb.FunctionConfig_Grpc{
				Grpc: &pb.GrpcFunctionConfig{
					Host: mmfTc.GetHostname(),
					Port: int32(mmfTc.GetGRPCPort()),
				},
			},
		},
		// TODO: Uncomment when rest config logic is implemented
		// &pb.FunctionConfig{
		// 	Name: mmfName,
		// 	Type: &pb.FunctionConfig_Rest{
		// 		Rest: &pb.RestFunctionConfig{
		// 			Host: mmfHost,
		// 			Port: mmfHTTPPortInt,
		// 		},
		// 	},
		// },
	}

	for _, fc := range fcs {
		validateFetchMatchesResult(assert, poolTickets, testProfiles, minimatchTc, fc)
	}
}

func validateFetchMatchesResult(assert *assert.Assertions, poolTickets map[string][]string, testProfiles []testProfile, tc *rpcTesting.TestContext, fc *pb.FunctionConfig) {
	be := pb.NewBackendClient(tc.MustGRPC())

	// Fetch Matches for each test profile.
	for _, profile := range testProfiles {
		brs, err := be.FetchMatches(tc.Context(), &pb.FetchMatchesRequest{
			Config:  fc,
			Profile: []*pb.MatchProfile{{Name: profile.name, Pool: profile.pools}},
		})
		assert.Nil(err)

		for {
			br, err := brs.Recv()
			if err == io.EOF {
				break
			}

			assert.Nil(err)
			assert.NotNil(br)
			assert.NotNil(br.Match)

			// Currently, the MMF simply creates a match per pool in the match profile - and populates
			// the roster with the pool name. Thus validate that for the roster populated in the match
			// result has all the tickets expected in that pool.
			assert.Equal(len(br.Match.Roster), 1)
			assert.Equal(br.Match.Roster[0].TicketId, poolTickets[br.Match.Roster[0].Name])

			var gotTickets []string
			for _, ticket := range br.Match.Ticket {
				gotTickets = append(gotTickets, ticket.Id)
			}

			// Given that currently we only populate all tickets in a match in a Roster, validate that
			// all the tickets present in the result match are equal to the tickets in the pool for that match.
			assert.Equal(gotTickets, poolTickets[br.Match.Roster[0].Name])
		}
	}
}

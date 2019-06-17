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
	"context"
	"io"
	"math"
	"testing"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/app/backend"
	"open-match.dev/open-match/internal/app/frontend"
	"open-match.dev/open-match/internal/app/mmlogic"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
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

func TestMinimatch(t *testing.T) {
	assert := require.New(t)
	minimatchTc := createMinimatchForTest(t)
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
			Type: &pb.FunctionConfig_Grpc{
				Grpc: &pb.GrpcFunctionConfig{
					Host: mmfTc.GetHostname(),
					Port: int32(mmfTc.GetGRPCPort()),
				},
			},
		},
		{
			Type: &pb.FunctionConfig_Rest{
				Rest: &pb.RestFunctionConfig{
					Host: mmfTc.GetHostname(),
					Port: int32(mmfTc.GetHTTPPort()),
				},
			},
		},
	}

	for _, fc := range fcs {
		testFetchMatches(assert, poolTickets, testProfiles, minimatchTc, be, fc)
	}
}

func testFetchMatches(assert *require.Assertions, poolTickets map[string][]string, testProfiles []testProfile, tc *rpcTesting.TestContext, be pb.BackendClient, fc *pb.FunctionConfig) {
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

// TODO: refactor the tests below this line
func TestAssignTickets(t *testing.T) {
	assert := assert.New(t)
	tc := createBackendForTest(t)
	defer tc.Close()

	fe := pb.NewFrontendClient(tc.MustGRPC())
	be := pb.NewBackendClient(tc.MustGRPC())

	ctResp, err := fe.CreateTicket(tc.Context(), &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	assert.Nil(err)

	var tt = []struct {
		req  *pb.AssignTicketsRequest
		resp *pb.AssignTicketsResponse
		code codes.Code
	}{
		{
			&pb.AssignTicketsRequest{},
			nil,
			codes.InvalidArgument,
		},
		{
			&pb.AssignTicketsRequest{
				TicketId: []string{"1"},
			},
			nil,
			codes.InvalidArgument,
		},
		{
			&pb.AssignTicketsRequest{
				TicketId: []string{"2"},
				Assignment: &pb.Assignment{
					Connection: "localhost",
				},
			},
			nil,
			codes.NotFound,
		},
		{
			&pb.AssignTicketsRequest{
				TicketId: []string{ctResp.Ticket.Id},
				Assignment: &pb.Assignment{
					Connection: "localhost",
				},
			},
			&pb.AssignTicketsResponse{},
			codes.OK,
		},
	}

	for _, test := range tt {
		resp, err := be.AssignTickets(tc.Context(), test.req)
		assert.Equal(test.resp, resp)
		if err != nil {
			assert.Equal(test.code, status.Convert(err).Code())
		} else {
			gtResp, err := fe.GetTicket(tc.Context(), &pb.GetTicketRequest{TicketId: ctResp.Ticket.Id})
			assert.Nil(err)
			assert.Equal(test.req.Assignment.Connection, gtResp.Assignment.Connection)
		}
	}
}

func TestFetchMatches(t *testing.T) {
	assert := assert.New(t)
	tc := createBackendForTest(t)
	defer tc.Close()

	be := pb.NewBackendClient(tc.MustGRPC())

	var tt = []struct {
		req  *pb.FetchMatchesRequest
		resp *pb.FetchMatchesResponse
		code codes.Code
	}{
		{
			&pb.FetchMatchesRequest{},
			nil,
			codes.InvalidArgument,
		},
	}

	for _, test := range tt {
		fetchMatchesLoop(t, tc, be, test.req, func(_ *pb.FetchMatchesResponse, err error) {
			assert.Equal(test.code, status.Convert(err).Code())
		})
	}
}

// TODO: Add FetchMatchesNormalTest when Hostname getter used to initialize mmf service is in
// https://github.com/GoogleCloudPlatform/open-match/pull/473
func fetchMatchesLoop(t *testing.T, tc *rpcTesting.TestContext, be pb.BackendClient, req *pb.FetchMatchesRequest, handleResponse func(*pb.FetchMatchesResponse, error)) {
	stream, err := be.FetchMatches(tc.Context(), req)
	if err != nil {
		t.Fatalf("error querying tickets, %v", err)
	}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		handleResponse(resp, err)
		if err != nil {
			return
		}
	}
}

func createBackendForTest(t *testing.T) *rpcTesting.TestContext {
	var closerFunc func()
	tc := rpcTesting.MustServe(t, func(p *rpc.ServerParams) {
		cfg := viper.New()
		closerFunc = statestoreTesting.New(t, cfg)

		cfg.Set("storage.page.size", 10)
		backend.BindService(p, cfg)
		frontend.BindService(p, cfg)
		mmlogic.BindService(p, cfg)
	})
	// TODO: This is very ugly. Need a better story around closing resources.
	tc.AddCloseFunc(closerFunc)
	return tc
}

// validateTicket validates that the fetched ticket is identical to the expected ticket.
func validateTicket(t *testing.T, got *pb.Ticket, want *pb.Ticket) {
	assert.Equal(t, got.Id, want.Id)
	assert.Equal(t, got.Properties.Fields["test-property"].GetNumberValue(), want.Properties.Fields["test-property"].GetNumberValue())
	assert.Equal(t, got.Assignment.Connection, want.Assignment.Connection)
	assert.Equal(t, got.Assignment.Properties, want.Assignment.Properties)
	assert.Equal(t, got.Assignment.Error, want.Assignment.Error)
}

// validateDelete validates that the ticket is actually deleted from the state storage.
// Given that delete is async, this method retries fetch every 100ms up to 5 seconds.
func validateDelete(t *testing.T, fe pb.FrontendClient, id string) {
	start := time.Now()
	for {
		if time.Since(start) > 5*time.Second {
			break
		}

		// Attempt to fetch the ticket every 100ms
		_, err := fe.GetTicket(context.Background(), &pb.GetTicketRequest{TicketId: id})
		if err != nil {
			// Only failure to fetch with NotFound should be considered as success.
			assert.Equal(t, status.Code(err), codes.NotFound)
			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	assert.Failf(t, "ticket %v not deleted after 5 seconds", id)
}

// TestFrontendService tests creating, getting and deleting a ticket using Frontend service.
func TestFrontendService(t *testing.T) {
	assert := assert.New(t)

	tc := createStore(t)
	fe := pb.NewFrontendClient(tc.MustGRPC())
	assert.NotNil(fe)

	ticket := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"test-property": {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
			},
		},
		Assignment: &pb.Assignment{
			Connection: "test-tbd",
		},
	}

	// Create a ticket, validate that it got an id and set its id in the expected ticket.
	resp, err := fe.CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: ticket})
	assert.NotNil(resp)
	assert.Nil(err)
	want := resp.Ticket
	assert.NotNil(want)
	assert.NotNil(want.Id)
	ticket.Id = want.Id
	validateTicket(t, resp.Ticket, ticket)

	// Fetch the ticket and validate that it is identical to the expected ticket.
	gotTicket, err := fe.GetTicket(context.Background(), &pb.GetTicketRequest{TicketId: ticket.Id})
	assert.NotNil(gotTicket)
	assert.Nil(err)
	validateTicket(t, gotTicket, ticket)

	// Delete the ticket and validate that it was actually deleted.
	_, err = fe.DeleteTicket(context.Background(), &pb.DeleteTicketRequest{TicketId: ticket.Id})
	assert.Nil(err)
	validateDelete(t, fe, ticket.Id)
}

func createStore(t *testing.T) *rpcTesting.TestContext {
	var closer func()
	tc := rpcTesting.MustServe(t, func(p *rpc.ServerParams) {
		cfg := viper.New()
		cfg.Set("playerIndices", []string{"testindex1", "testindex2"})
		closer = statestoreTesting.New(t, cfg)
		err := frontend.BindService(p, cfg)
		assert.Nil(t, err)
	})
	tc.AddCloseFunc(closer)
	return tc
}

// TODO: move below test cases to e2e
func TestQueryTicketsEmptyRequest(t *testing.T) {
	assert := assert.New(t)
	tc := createMmlogicForTest(t)
	defer tc.Close()

	queryTicketsLoop(t, tc, &pb.QueryTicketsRequest{}, func(_ *pb.QueryTicketsResponse, err error) {
		assert.Equal(codes.InvalidArgument, status.Convert(err).Code())
	})
}

func TestQueryTicketsForEmptyDatabase(t *testing.T) {
	assert := assert.New(t)
	tc := createMmlogicForTest(t)
	defer tc.Close()

	queryTicketsLoop(t, tc,
		&pb.QueryTicketsRequest{
			Pool: &pb.Pool{
				Filter: []*pb.Filter{{
					Attribute: "ok",
				}},
			},
		},
		func(resp *pb.QueryTicketsResponse, err error) {
			assert.NotNil(resp)
		})
}

func queryTicketsLoop(t *testing.T, tc *rpcTesting.TestContext, req *pb.QueryTicketsRequest, handleResponse func(*pb.QueryTicketsResponse, error)) {
	c := pb.NewMmLogicClient(tc.MustGRPC())
	stream, err := c.QueryTickets(tc.Context(), req)
	if err != nil {
		t.Fatalf("error querying tickets, %v", err)
	}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		handleResponse(resp, err)
		if err != nil {
			return
		}
	}
}

func createMmlogicForTest(t *testing.T) *rpcTesting.TestContext {
	var closerFunc func()
	tc := rpcTesting.MustServe(t, func(p *rpc.ServerParams) {
		cfg := viper.New()
		closerFunc = statestoreTesting.New(t, cfg)
		cfg.Set("storage.page.size", 10)

		mmlogic.BindService(p, cfg)
	})
	// TODO: This is very ugly. Need a better story around closing resources.
	tc.AddCloseFunc(closerFunc)
	return tc
}

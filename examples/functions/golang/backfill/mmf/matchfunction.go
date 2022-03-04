// Copyright 2020 Google LLC
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

// Package mmf provides a sample match function that uses the GRPC harness to set up 1v1 matches.
// This sample is a reference to demonstrate the usage of backfill and should only be used as
// a starting point for your match function. You will need to modify the
// matchmaking logic in this function based on your game's requirements.
package mmf

import (
	"fmt"
	"time"

	"log"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"open-match.dev/open-match/pkg/matchfunction"
	"open-match.dev/open-match/pkg/pb"
)

const (
	playersPerMatch = 2
	openSlotsKey    = "open-slots"
	matchName       = "backfill-matchfunction"
)

// matchFunctionService implements pb.MatchFunctionServer, the server generated
// by compiling the protobuf, by fulfilling the pb.MatchFunctionServer interface.
type matchFunctionService struct {
	grpc               *grpc.Server
	queryServiceClient pb.QueryServiceClient
	port               int
}

func (s *matchFunctionService) Run(req *pb.RunRequest, stream pb.MatchFunction_RunServer) error {
	log.Printf("Generating proposals for function %v", req.GetProfile().GetName())

	var proposals []*pb.Match
	profile := req.GetProfile()
	pools := profile.GetPools()

	for _, p := range pools {
		tickets, err := matchfunction.QueryPool(stream.Context(), s.queryServiceClient, p)
		if err != nil {
			log.Printf("Failed to query tickets for the given pool, got %s", err.Error())
			return err
		}

		backfills, err := matchfunction.QueryBackfillPool(stream.Context(), s.queryServiceClient, p)
		if err != nil {
			log.Printf("Failed to query backfills for the given pool, got %s", err.Error())
			return err
		}

		matches, err := makeMatches(profile, p, tickets, backfills)
		if err != nil {
			log.Printf("Failed to generate matches, got %s", err.Error())
			return err
		}

		proposals = append(proposals, matches...)
	}

	log.Printf("Streaming %v proposals to Open Match", len(proposals))
	// Stream the generated proposals back to Open Match.
	for _, proposal := range proposals {
		if err := stream.Send(&pb.RunResponse{Proposal: proposal}); err != nil {
			log.Printf("Failed to stream proposals to Open Match, got %s", err.Error())
			return err
		}
	}

	return nil
}

// makeMatches tries to handle backfills at first, then it makes full matches, at the end it makes a match with backfill
// if tickets left
func makeMatches(profile *pb.MatchProfile, pool *pb.Pool, tickets []*pb.Ticket, backfills []*pb.Backfill) ([]*pb.Match, error) {
	var matches []*pb.Match
	newMatches, remainingTickets, err := handleBackfills(profile, tickets, backfills, len(matches))
	if err != nil {
		return nil, err
	}

	matches = append(matches, newMatches...)
	newMatches, remainingTickets = makeFullMatches(profile, remainingTickets, len(matches))
	matches = append(matches, newMatches...)

	if len(remainingTickets) > 0 {
		match, err := makeMatchWithBackfill(profile, pool, remainingTickets, len(matches))
		if err != nil {
			return nil, err
		}

		matches = append(matches, match)
	}

	return matches, nil
}

// handleBackfills looks at each backfill's openSlots which is a number of required tickets,
// acquires that tickets, decreases openSlots in backfill and makes a match with updated backfill and associated tickets.
func handleBackfills(profile *pb.MatchProfile, tickets []*pb.Ticket, backfills []*pb.Backfill, lastMatchId int) ([]*pb.Match, []*pb.Ticket, error) {
	matchId := lastMatchId
	var matches []*pb.Match

	for _, b := range backfills {
		openSlots, err := getOpenSlots(b)
		if err != nil {
			return nil, tickets, err
		}

		var matchTickets []*pb.Ticket
		for openSlots > 0 && len(tickets) > 0 {
			matchTickets = append(matchTickets, tickets[0])
			tickets = tickets[1:]
			openSlots--
		}

		if len(matchTickets) > 0 {
			err := setOpenSlots(b, openSlots)
			if err != nil {
				return nil, tickets, err
			}

			matchId++
			match := newMatch(matchId, profile.Name, matchTickets, b)
			matches = append(matches, &match)
		}
	}

	return matches, tickets, nil
}

// makeMatchWithBackfill makes not full match, creates backfill for it with openSlots = playersPerMatch-len(tickets).
func makeMatchWithBackfill(profile *pb.MatchProfile, pool *pb.Pool, tickets []*pb.Ticket, lastMatchId int) (*pb.Match, error) {
	if len(tickets) == 0 {
		return nil, fmt.Errorf("tickets are required")
	}

	if len(tickets) >= playersPerMatch {
		return nil, fmt.Errorf("too many tickets")
	}

	matchId := lastMatchId
	searchFields := newSearchFields(pool)
	backfill, err := newBackfill(searchFields, playersPerMatch-len(tickets))
	if err != nil {
		return nil, err
	}

	matchId++
	match := newMatch(matchId, profile.Name, tickets, backfill)
	// indicates that it is a new match and new game server should be allocated for it
	match.AllocateGameserver = true

	return &match, nil
}

// makeFullMatches makes matches without backfill
func makeFullMatches(profile *pb.MatchProfile, tickets []*pb.Ticket, lastMatchId int) ([]*pb.Match, []*pb.Ticket) {
	ticketNum := 0
	matchId := lastMatchId
	var matches []*pb.Match

	for ticketNum < playersPerMatch && len(tickets) >= playersPerMatch {
		ticketNum++

		if ticketNum == playersPerMatch {
			matchId++

			match := newMatch(matchId, profile.Name, tickets[:playersPerMatch], nil)
			matches = append(matches, &match)

			tickets = tickets[playersPerMatch:]
			ticketNum = 0
		}
	}

	return matches, tickets
}

// newSearchFields creates search fields based on pool's search criteria. This is just example of how it can be done.
func newSearchFields(pool *pb.Pool) *pb.SearchFields {
	searchFields := pb.SearchFields{}
	rangeFilters := pool.GetDoubleRangeFilters()

	if rangeFilters != nil {
		doubleArgs := make(map[string]float64)
		for _, f := range rangeFilters {
			doubleArgs[f.DoubleArg] = (f.Max - f.Min) / 2
		}

		if len(doubleArgs) > 0 {
			searchFields.DoubleArgs = doubleArgs
		}
	}

	stringFilters := pool.GetStringEqualsFilters()

	if stringFilters != nil {
		stringArgs := make(map[string]string)
		for _, f := range stringFilters {
			stringArgs[f.StringArg] = f.Value
		}

		if len(stringArgs) > 0 {
			searchFields.StringArgs = stringArgs
		}
	}

	tagFilters := pool.GetTagPresentFilters()

	if tagFilters != nil {
		tags := make([]string, len(tagFilters))
		for _, f := range tagFilters {
			tags = append(tags, f.Tag)
		}

		if len(tags) > 0 {
			searchFields.Tags = tags
		}
	}

	return &searchFields
}

func newBackfill(searchFields *pb.SearchFields, openSlots int) (*pb.Backfill, error) {
	b := pb.Backfill{
		SearchFields: searchFields,
		Generation:   0,
		CreateTime:   timestamppb.Now(),
	}

	err := setOpenSlots(&b, int32(openSlots))
	return &b, err
}

func newMatch(num int, profile string, tickets []*pb.Ticket, b *pb.Backfill) pb.Match {
	t := time.Now().Format("2006-01-02T15:04:05.00")

	return pb.Match{
		MatchId:       fmt.Sprintf("profile-%s-time-%s-num-%d", profile, t, num),
		MatchProfile:  profile,
		MatchFunction: matchName,
		Tickets:       tickets,
		Backfill:      b,
	}
}

func setOpenSlots(b *pb.Backfill, val int32) error {
	if b.Extensions == nil {
		b.Extensions = make(map[string]*anypb.Any)
	}

	any, err := anypb.New(&wrapperspb.Int32Value{Value: val})
	if err != nil {
		return err
	}

	b.Extensions[openSlotsKey] = any
	return nil
}

func getOpenSlots(b *pb.Backfill) (int32, error) {
	if b == nil {
		return 0, fmt.Errorf("expected backfill is not nil")
	}

	if b.Extensions != nil {
		if any, ok := b.Extensions[openSlotsKey]; ok {
			var val wrapperspb.Int32Value
			err := any.UnmarshalTo(&val)
			if err != nil {
				return 0, err
			}

			return val.Value, nil
		}
	}

	return playersPerMatch, nil
}

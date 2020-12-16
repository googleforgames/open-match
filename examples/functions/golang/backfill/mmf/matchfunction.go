package mmf

import (
	"fmt"
	"time"

	"log"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"google.golang.org/grpc"
	"open-match.dev/open-match/pkg/matchfunction"
	"open-match.dev/open-match/pkg/pb"
)

const (
	playersPerMatch = 2
	extensionsKey   = "backfill-extensions"
	matchName       = "backfill-matchfunction"
)

type matchFunctionService struct {
	grpc               *grpc.Server
	queryServiceClient pb.QueryServiceClient
	port               int
}

func (s *matchFunctionService) Run(req *pb.RunRequest, stream pb.MatchFunction_RunServer) error {
	log.Printf("Generating proposals for function %v", req.GetProfile().GetName())

	var proposals []*pb.Match
	pools := req.GetProfile().GetPools()

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

		matches, err := makeMatches(p, tickets, backfills)
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

func makeMatches(pool *pb.Pool, tickets []*pb.Ticket, backfills []*pb.Backfill) ([]*pb.Match, error) {
	var matches []*pb.Match
	newMatches, remainingTickets, err := handleBackfills(tickets, backfills, len(matches))
	if err != nil {
		return nil, err
	}

	matches = append(matches, newMatches...)
	newMatches, remainingTickets = makeFullMatches(remainingTickets, len(matches))
	matches = append(matches, newMatches...)

	if len(remainingTickets) > 0 {
		match, err := makeMatchWithBackfill(pool, remainingTickets, len(matches))
		if err != nil {
			return nil, err
		}

		matches = append(matches, match)
	}

	return matches, nil
}

func handleBackfills(tickets []*pb.Ticket, backfills []*pb.Backfill, lastMatchId int) ([]*pb.Match, []*pb.Ticket, error) {
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
			match := newMatch(matchId, matchTickets, b)
			matches = append(matches, &match)
		}
	}

	return matches, tickets, nil
}

func makeMatchWithBackfill(pool *pb.Pool, tickets []*pb.Ticket, lastMatchId int) (*pb.Match, error) {
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
	match := newMatch(matchId, tickets, backfill)
	match.AllocateGameserver = true

	return &match, nil
}

func makeFullMatches(tickets []*pb.Ticket, lastMatchId int) ([]*pb.Match, []*pb.Ticket) {
	ticketNum := 0
	matchId := lastMatchId
	var matches []*pb.Match

	for ticketNum < playersPerMatch && len(tickets) >= playersPerMatch {
		ticketNum++

		if ticketNum == playersPerMatch {
			matchId++

			match := newMatch(matchId, tickets[:playersPerMatch], nil)
			matches = append(matches, &match)

			tickets = tickets[playersPerMatch:]
			ticketNum = 0
		}
	}

	return matches, tickets
}

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
		CreateTime:   ptypes.TimestampNow(),
	}

	err := setOpenSlots(&b, int32(openSlots))
	return &b, err
}

func newMatch(num int, tickets []*pb.Ticket, b *pb.Backfill) pb.Match {
	t := time.Now().Format("2006-01-02T15:04:05.00")

	return pb.Match{
		MatchId:       fmt.Sprintf("profile-%s-time-%s-num-%d", matchName, t, num),
		MatchProfile:  matchName,
		MatchFunction: matchName,
		Tickets:       tickets,
		Backfill:      b,
	}
}

func setOpenSlots(b *pb.Backfill, val int32) error {
	if b.Extensions == nil {
		b.Extensions = make(map[string]*any.Any)
	}

	any, err := ptypes.MarshalAny(&BackfillExtensions{
		OpenSlots: val,
	})
	if err != nil {
		return err
	}

	b.Extensions[extensionsKey] = any
	return nil
}

func getOpenSlots(b *pb.Backfill) (int32, error) {
	if b == nil {
		return 0, fmt.Errorf("expected backfill is not nil")
	}

	if b.Extensions != nil {
		if any, ok := b.Extensions[extensionsKey]; ok {
			var ext BackfillExtensions
			err := ptypes.UnmarshalAny(any, &ext)
			if err != nil {
				return 0, err
			}

			return ext.OpenSlots, nil
		}
	}

	return playersPerMatch, nil
}

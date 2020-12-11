package mmf

import (
	"testing"

	"github.com/stretchr/testify/require"
	"open-match.dev/open-match/pkg/pb"
)

func TestHandleBackfills(t *testing.T) {
	for _, tc := range []struct {
		name              string
		tickets           []*pb.Ticket
		backfills         []*pb.Backfill
		lastMatchId       int
		expectedMatchLen  int
		expectedTicketLen int
		expectedOpenSlots int32
		expectedErr       bool
	}{
		{name: "returns no matches when no backfills specified", expectedMatchLen: 0, expectedTicketLen: 0},
		{name: "returns no matches when no tickets specified", expectedMatchLen: 0, expectedTicketLen: 0},
		{name: "returns a match with open slots decreased", tickets: []*pb.Ticket{{Id: "1"}}, backfills: []*pb.Backfill{{}}, expectedMatchLen: 1, expectedTicketLen: 0, expectedOpenSlots: playersPerMatch - 1},
	} {
		testCase := tc
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			matches, tickets, err := handleBackfills(testCase.tickets, testCase.backfills, testCase.lastMatchId)
			require.Equal(t, testCase.expectedErr, err != nil)
			require.Equal(t, testCase.expectedTicketLen, len(tickets))

			if err != nil {
				require.Equal(t, 0, len(matches))
			} else {
				for _, m := range matches {
					require.NotNil(t, m.Backfill)

					openSlots, err := getOpenSlots(m.Backfill)
					require.NoError(t, err)
					require.Equal(t, testCase.expectedOpenSlots, openSlots)
				}
			}
		})
	}
}

func TestMakeMatchWithBackfill(t *testing.T) {
	for _, testCase := range []struct {
		name              string
		tickets           []*pb.Ticket
		lastMatchId       int
		expectedOpenSlots int32
		expectedErr       bool
	}{
		{name: "returns an error when length of tickets is greater then playerPerMatch", tickets: []*pb.Ticket{{Id: "1"}, {Id: "2"}, {Id: "3"}, {Id: "4"}, {Id: "5"}}, expectedErr: true},
		{name: "returns an error when length of tickets is equal to playerPerMatch", tickets: []*pb.Ticket{{Id: "1"}, {Id: "2"}, {Id: "3"}, {Id: "4"}}, expectedErr: true},
		{name: "returns an error when no tickets are provided", expectedErr: true},
		{name: "returns a match with backfill", tickets: []*pb.Ticket{{Id: "1"}}, expectedOpenSlots: playersPerMatch - 1},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pool := pb.Pool{}
			match, err := makeMatchWithBackfill(&pool, testCase.tickets, testCase.lastMatchId)
			require.Equal(t, testCase.expectedErr, err != nil)

			if err == nil {
				require.NotNil(t, match)
				require.NotNil(t, match.Backfill)
				require.True(t, match.AllocateGameserver)
				require.Equal(t, "", match.Backfill.Id)

				openSlots, err := getOpenSlots(match.Backfill)
				require.Nil(t, err)
				require.Equal(t, testCase.expectedOpenSlots, openSlots)
			}
		})

	}
}

func TestMakeFullMatches(t *testing.T) {
	for _, testCase := range []struct {
		name              string
		tickets           []*pb.Ticket
		lastMatchId       int
		expectedMatchLen  int
		expectedTicketLen int
	}{
		{name: "returns no matches when there are no tickets", tickets: []*pb.Ticket{}, expectedMatchLen: 0, expectedTicketLen: 0},
		{name: "returns no matches when lenght of tickets is less then playersPerMatch", tickets: []*pb.Ticket{{Id: "1"}}, expectedMatchLen: 0, expectedTicketLen: 1},
		{name: "returns a match when length of tickets is greater then playersPerMatch", tickets: []*pb.Ticket{{Id: "1"}, {Id: "2"}}, expectedMatchLen: 1, expectedTicketLen: 0},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			matches, tickets := makeFullMatches(testCase.tickets, testCase.lastMatchId)

			require.Equal(t, testCase.expectedMatchLen, len(matches))
			require.Equal(t, testCase.expectedTicketLen, len(tickets))

			for _, m := range matches {
				require.Nil(t, m.Backfill)
				require.Equal(t, playersPerMatch, len(m.Tickets))
			}
		})
	}
}

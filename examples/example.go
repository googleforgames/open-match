package example

import (
	"open-match.dev/open-match/pkg/pb"
)

func main() {

	// To create a ticket, use SearchFields instead of using Properties
	ticket := &pb.Ticket{
		// Includes all fields that need to be indexed
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{"defense": 10, "level": 50},
			StringArgs: map[string]string{"player-signature": "nice to see you"},
			Tags:       []string{"beta-gameplay", "demo-map"},
		},
	}

	fmReq := &pb.FetchMatchesRequest{
		Config: &pb.FunctionConfig{}, // Unchanged
		MatchProfiles: []*pb.MatchProfile{
			{
				Name: "this is a profile", // Unchanged
				Pools: []*pb.Pool{
					{
						Name: "this is a pool", // Unchanged
						DoubleRangeFilters: []*pb.DoubleRangeFilter{
							{
								DoubleArg: "defense",
								Min:       0,
								Max:       math.MaxFloat64,
							},
						},
						StringEqualsFilters: []*pb.StringEqualsFilter{
							{
								StringArg: "player-signature",
								Value:     "nice to see you",
							},
						},
						TagPresentFilters: []*pb.TagPresentFilter{
							{
								Tag: "demo-map",
							},
						},
					},
				},
				Rosters: []*pb.Roster{}, // Unchanged
			},
		},
	}

}

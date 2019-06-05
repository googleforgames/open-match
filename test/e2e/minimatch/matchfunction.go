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
	"github.com/rs/xid"
	"github.com/spf13/viper"
	harness "open-match.dev/open-match/internal/harness/golang"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"
)

const (
	mmfName        = "test-matchfunction"
	mmfPrefix      = "testmmf"
	mmfHost        = "localhost"
	mmfGRPCPort    = "50511"
	mmfHTTPPort    = "51511"
	mmfGRPCPortInt = 50511
	mmfHTTPPortInt = 51511
)

// serveMatchFunction creates a GRPC server and starts it to server the match function forever.
func serveMatchFunction() (func(), error) {
	cfg := viper.New()
	// Set match function configuration.
	cfg.Set("testmmf.hostname", mmfHost)
	cfg.Set("testmmf.grpcport", mmfGRPCPort)
	cfg.Set("testmmf.httpport", mmfHTTPPort)

	// The below configuration is used by GRPC harness to create an mmlogic client to query tickets.
	cfg.Set("api.mmlogic.hostname", minimatchHost)
	cfg.Set("api.mmlogic.grpcport", minimatchGRPCPort)
	cfg.Set("api.mmlogic.httpport", minimatchHTTPPort)

	p, err := rpc.NewServerParamsFromConfig(cfg, mmfPrefix)
	if err != nil {
		return nil, err
	}

	if err := harness.BindService(p, cfg, &harness.FunctionSettings{
		FunctionName: mmfName,
		Func:         makeMatches,
	}); err != nil {
		return nil, err
	}

	s := &rpc.Server{}
	waitForStart, err := s.Start(p)
	if err != nil {
		return nil, err
	}
	go func() {
		waitForStart()
	}()

	return func() { s.Stop() }, nil
}

// This is the core match making function that will be triggered by Open Match to generate matches.
// The goal of this function is to generate predictable matches that can be validated without flakyness.
// This match function loops through all the pools and generates one match per pool aggregating all players
// in that pool in the generated match.
func makeMatches(params *harness.MatchFunctionParams) []*pb.Match {
	var result []*pb.Match
	for pool, tickets := range params.PoolNameToTickets {
		roster := &pb.Roster{Name: pool}
		for _, ticket := range tickets {
			roster.TicketId = append(roster.TicketId, ticket.Id)
		}

		result = append(result, &pb.Match{
			MatchId:       xid.New().String(),
			MatchProfile:  params.ProfileName,
			MatchFunction: mmfName,
			Ticket:        tickets,
			Roster:        []*pb.Roster{roster},
		})
	}

	return result
}

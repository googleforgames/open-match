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

// Package matchfunction provides helper methods to simplify authoring a match fuction.
package matchfunction

import (
	"context"
	"fmt"
	"io"

	"open-match.dev/open-match/pkg/pb"
)

// QueryPool queries queryService and returns the tickets that belong to the specified pool.
func QueryPool(ctx context.Context, mml pb.QueryServiceClient, pool *pb.Pool) ([]*pb.Ticket, error) {
	query, err := mml.QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: pool})
	if err != nil {
		return nil, fmt.Errorf("Error calling queryService.QueryTickets: %w", err)
	}

	var tickets []*pb.Ticket
	for {
		resp, err := query.Recv()
		if err == io.EOF {
			return tickets, nil
		}

		if err != nil {
			return nil, fmt.Errorf("Error receiving tickets from queryService.QueryTickets: %w", err)
		}

		tickets = append(tickets, resp.Tickets...)
	}
}

// QueryPools queries queryService and returns the a map of pool names to the tickets belonging to those pools.
func QueryPools(ctx context.Context, mml pb.QueryServiceClient, pools []*pb.Pool) (map[string][]*pb.Ticket, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	type result struct {
		err     error
		tickets []*pb.Ticket
		name    string
	}

	results := make(chan result)
	for _, pool := range pools {
		go func(pool *pb.Pool) {
			r := result{
				name: pool.Name,
			}
			r.tickets, r.err = QueryPool(ctx, mml, pool)
			select {
			case results <- r:
			case <-ctx.Done():
			}
		}(pool)
	}

	poolMap := make(map[string][]*pb.Ticket)
	for i := 0; i < len(pools); i++ {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("Context canceled while querying pools: %w", ctx.Err())
		case r := <-results:
			if r.err != nil {
				return nil, r.err
			}

			poolMap[r.name] = r.tickets
		}
	}

	return poolMap, nil
}

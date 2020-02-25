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

package query

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/filter"
	"open-match.dev/open-match/internal/statestore"
	"open-match.dev/open-match/pkg/pb"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.query",
	})
)

// queryService API provides utility functions for common MMF functionality such
// as retreiving Tickets from state storage.
type queryService struct {
	cfg           config.View
	store         statestore.Service
	queryRequests chan *queryRequest

	// Single item buffered channel.  Holds a value when runQuery can be safely
	// started.  Basically a channel/select friendly mutex around runQuery
	// running.
	canStartRunQuery chan struct{}

	// NOT multithread safe, only to be modified by runQuery, and read when
	// holding a waitgroup created by runQuery.
	tc *ticketCache
}

func newQueryService(cfg config.View) *queryService {
	return &queryService{
		cfg:           cfg,
		store:         statestore.New(cfg),
		queryRequests: make(chan *queryRequest),
		tc: &ticketCache{
			listed: make(map[string]*pb.Ticket),
		},
	}
}

type queryRequest struct {
	ctx  context.Context
	resp chan *sync.WaitGroup
}

// QueryTickets gets a list of Tickets that match all Filters of the input Pool.
//   - If the Pool contains no Filters, QueryTickets will return all Tickets in the state storage.
// QueryTickets pages the Tickets by `storage.pool.size` and stream back response.
//   - storage.pool.size is default to 1000 if not set, and has a mininum of 10 and maximum of 10000
func (s *queryService) QueryTickets(req *pb.QueryTicketsRequest, responseServer pb.QueryService_QueryTicketsServer) error {
	pool := req.GetPool()
	if pool == nil {
		return status.Error(codes.InvalidArgument, ".pool is required")
	}

	ctx := responseServer.Context()
	pSize := getPageSize(s.cfg)

	qr := &queryRequest{
		ctx:  ctx,
		resp: make(chan *sync.WaitGroup),
	}

sendRequest:
	for {
		select {
		case <-ctx.Done():
			logger.Errorf("QueryTickets canceled before request sent.")
			return ctx.Err()
		case <-s.canStartRunQuery:
			go s.runQuery()
		case s.queryRequests <- qr:
			break sendRequest
		}
	}

	var wg *sync.WaitGroup

	select {
	case <-ctx.Done():
		logger.Errorf("QueryTickets canceled waiting for access.")
		return ctx.Err()
	case wg = <-qr.resp:
	}

	logger.Errorf("QueryTickets ran query")
	tickets, err := s.tc.query(pool)
	wg.Done()

	if err != nil {
		return err
	}
	for start := 0; start < len(tickets); start += pSize {
		end := start + pSize
		if end > len(tickets) {
			end = len(tickets)
		}

		err := responseServer.Send(&pb.QueryTicketsResponse{
			Tickets: tickets[start:end],
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func getPageSize(cfg config.View) int {
	const (
		name = "storage.page.size"
		// Minimum number of tickets to be returned in a streamed response for QueryTickets. This value
		// will be used if page size is configured lower than the minimum value.
		minPageSize int = 10
		// Default number of tickets to be returned in a streamed response for QueryTickets.  This value
		// will be used if page size is not configured.
		defaultPageSize int = 1000
		// Maximum number of tickets to be returned in a streamed response for QueryTickets. This value
		// will be used if page size is configured higher than the maximum value.
		maxPageSize int = 10000
	)

	if !cfg.IsSet(name) {
		return defaultPageSize
	}

	pSize := cfg.GetInt("storage.page.size")
	if pSize < minPageSize {
		logger.Infof("page size %v is lower than the minimum limit of %v", pSize, maxPageSize)
		pSize = minPageSize
	}

	if pSize > maxPageSize {
		logger.Infof("page size %v is higher than the maximum limit of %v", pSize, maxPageSize)
		return maxPageSize
	}

	return pSize
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

type ticketCache struct {
	listed map[string]*pb.Ticket
	err    error
}

func (tc *ticketCache) query(pool *pb.Pool) ([]*pb.Ticket, error) {
	if tc.err != nil {
		return nil, tc.err
	}

	var results []*pb.Ticket

	for _, ticket := range tc.listed {
		if filter.InPool(ticket, pool) {
			results = append(results, ticket)
		}
	}

	return results, nil
}

// Does not fail, but may set error which will be returned to clients.
func (tc *ticketCache) update(store statestore.Service) {
	previousCount := len(tc.listed)

	currentAll, err := store.GetIndexedIds(context.Background())
	if err != nil {
		tc.err = err
		return
	}

	deletedCount := 0
	for id := range tc.listed {
		if _, ok := currentAll[id]; !ok {
			delete(tc.listed, id)
			deletedCount++
		}
	}

	toFetch := []string{}

	for id := range currentAll {
		if _, ok := tc.listed[id]; !ok {
			toFetch = append(toFetch, id)
		}
	}

	newTickets, err := store.GetTickets(context.Background(), toFetch)
	if err != nil {
		tc.err = err
		return
	}

	for _, t := range newTickets {
		tc.listed[t.Id] = t
	}

	logger.Warningf("Previous %d, Deleted %d, toFetch %d, Current %d", previousCount, deletedCount, len(toFetch), len(tc.listed))
	tc.err = nil
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (s *queryService) runQuery() {
	defer func() {
		s.canStartRunQuery <- struct{}{}
	}()

	// Wait for first query request.
	reqs := []*queryRequest{<-s.queryRequests}

	// Collect all waiting queries.
collectAllWaiting:
	for {
		select {
		case req := <-s.queryRequests:
			reqs = append(reqs, req)
		default:
			break collectAllWaiting
		}
	}

	s.tc.update(s.store)

	wg := &sync.WaitGroup{}

	// Send WaitGroup to query calls, letting them run their query on the ticket
	// cache.
	for _, req := range reqs {
		select {
		case req.resp <- wg:
			wg.Add(1)
		case <-req.ctx.Done():
		}
	}

	// wait for query calls to finish using ticket cache.
	wg.Wait()
}

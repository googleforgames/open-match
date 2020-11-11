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
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/appmain"
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
	cfg config.View
	tc  *ticketCache
	bc  *backfillCache
}

func (s *queryService) QueryTickets(req *pb.QueryTicketsRequest, responseServer pb.QueryService_QueryTicketsServer) error {
	ctx := responseServer.Context()
	pool := req.GetPool()
	if pool == nil {
		return status.Error(codes.InvalidArgument, ".pool is required")
	}

	pf, err := filter.NewPoolFilter(pool)
	if err != nil {
		return err
	}

	var results []*pb.Ticket
	err = s.tc.request(ctx, func(tickets map[string]*pb.Ticket) {
		for _, ticket := range tickets {
			if pf.In(ticket) {
				results = append(results, ticket)
			}
		}
	})
	if err != nil {
		err = errors.Wrap(err, "QueryTickets: failed to run request")
		return err
	}
	stats.Record(ctx, ticketsPerQuery.M(int64(len(results))))

	pSize := getPageSize(s.cfg)
	for start := 0; start < len(results); start += pSize {
		end := start + pSize
		if end > len(results) {
			end = len(results)
		}

		err := responseServer.Send(&pb.QueryTicketsResponse{
			Tickets: results[start:end],
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *queryService) QueryTicketIds(req *pb.QueryTicketIdsRequest, responseServer pb.QueryService_QueryTicketIdsServer) error {
	ctx := responseServer.Context()
	pool := req.GetPool()
	if pool == nil {
		return status.Error(codes.InvalidArgument, ".pool is required")
	}

	pf, err := filter.NewPoolFilter(pool)
	if err != nil {
		return err
	}

	var results []string
	err = s.tc.request(ctx, func(tickets map[string]*pb.Ticket) {
		for id, ticket := range tickets {
			if pf.In(ticket) {
				results = append(results, id)
			}
		}
	})
	if err != nil {
		err = errors.Wrap(err, "QueryTicketIds: failed to run request")
		return err
	}
	stats.Record(ctx, ticketsPerQuery.M(int64(len(results))))

	pSize := getPageSize(s.cfg)
	for start := 0; start < len(results); start += pSize {
		end := start + pSize
		if end > len(results) {
			end = len(results)
		}

		err := responseServer.Send(&pb.QueryTicketIdsResponse{
			Ids: results[start:end],
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *queryService) QueryBackfills(req *pb.QueryBackfillsRequest, responseServer pb.QueryService_QueryBackfillsServer) error {
	ctx := responseServer.Context()
	pool := req.GetPool()
	if pool == nil {
		return status.Error(codes.InvalidArgument, ".pool is required")
	}

	pf, err := filter.NewPoolFilter(pool)
	if err != nil {
		return err
	}

	var results []*pb.Backfill
	err = s.bc.request(ctx, func(backfills map[string]*pb.Backfill) {
		for _, backfill := range backfills {
			if pf.In(backfill) {
				results = append(results, backfill)
			}
		}
	})
	if err != nil {
		err = errors.Wrap(err, "QueryBackfills: failed to run request")
		return err
	}

	stats.Record(ctx, backfillsPerQuery.M(int64(len(results))))

	pSize := getPageSize(s.cfg)
	for start := 0; start < len(results); start += pSize {
		end := start + pSize
		if end > len(results) {
			end = len(results)
		}

		err := responseServer.Send(&pb.QueryBackfillsResponse{
			Backfills: results[start:end],
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func getPageSize(cfg config.View) int {
	const (
		name = "queryPageSize"
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

	pSize := cfg.GetInt(name)
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

/////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////

// ticketCache unifies concurrent requests into a single cache update, and
// gives a safe view into that map cache.
type ticketCache struct {
	store statestore.Service

	requests chan *cacheRequest

	// Single item buffered channel.  Holds a value when runQuery can be safely
	// started.  Basically a channel/select friendly mutex around runQuery
	// running.
	startRunRequest chan struct{}

	wg sync.WaitGroup

	// Mutlithreaded unsafe fields, only to be written by update, and read when
	// request given the ok.
	tickets map[string]*pb.Ticket
	err     error
}

func newTicketCache(b *appmain.Bindings, cfg config.View) *ticketCache {
	tc := &ticketCache{
		store:           statestore.New(cfg),
		requests:        make(chan *cacheRequest),
		startRunRequest: make(chan struct{}, 1),
		tickets:         make(map[string]*pb.Ticket),
	}

	tc.startRunRequest <- struct{}{}
	b.AddHealthCheckFunc(tc.store.HealthCheck)

	return tc
}

type cacheRequest struct {
	ctx    context.Context
	runNow chan struct{}
}

func (tc *ticketCache) request(ctx context.Context, f func(map[string]*pb.Ticket)) error {
	cr := &cacheRequest{
		ctx:    ctx,
		runNow: make(chan struct{}),
	}

sendRequest:
	for {
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "ticket cache request canceled before reuest sent.")
		case <-tc.startRunRequest:
			go tc.runRequest()
		case tc.requests <- cr:
			break sendRequest
		}
	}

	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "ticket cache request canceled waiting for access.")
	case <-cr.runNow:
		defer tc.wg.Done()
	}

	if tc.err != nil {
		return tc.err
	}

	f(tc.tickets)
	return nil
}

func (tc *ticketCache) runRequest() {
	defer func() {
		tc.startRunRequest <- struct{}{}
	}()

	// Wait for first query request.
	reqs := []*cacheRequest{<-tc.requests}

	// Collect all waiting queries.
collectAllWaiting:
	for {
		select {
		case req := <-tc.requests:
			reqs = append(reqs, req)
		default:
			break collectAllWaiting
		}
	}

	tc.update()
	stats.Record(context.Background(), cacheWaitingQueries.M(int64(len(reqs))))

	// Send WaitGroup to query calls, letting them run their query on the ticket
	// cache.
	for _, req := range reqs {
		select {
		case req.runNow <- struct{}{}:
			tc.wg.Add(1)
		case <-req.ctx.Done():
		}
	}

	// wait for requests to finish using ticket cache.
	tc.wg.Wait()
}

func (tc *ticketCache) update() {
	st := time.Now()
	previousCount := len(tc.tickets)

	currentAll, err := tc.store.GetIndexedIDSet(context.Background())
	if err != nil {
		tc.err = err
		return
	}

	deletedCount := 0
	for id := range tc.tickets {
		if _, ok := currentAll[id]; !ok {
			delete(tc.tickets, id)
			deletedCount++
		}
	}

	toFetch := []string{}

	for id := range currentAll {
		if _, ok := tc.tickets[id]; !ok {
			toFetch = append(toFetch, id)
		}
	}

	newTickets, err := tc.store.GetTickets(context.Background(), toFetch)
	if err != nil {
		tc.err = err
		return
	}

	for _, t := range newTickets {
		tc.tickets[t.Id] = t
	}

	stats.Record(context.Background(), cacheTotalItems.M(int64(previousCount)))
	stats.Record(context.Background(), cacheFetchedItems.M(int64(len(toFetch))))
	stats.Record(context.Background(), cacheUpdateLatency.M(float64(time.Since(st))/float64(time.Millisecond)))

	logger.Debugf("Ticket Cache update: Previous %d, Deleted %d, Fetched %d, Current %d", previousCount, deletedCount, len(toFetch), len(tc.tickets))
	tc.err = nil
}

type backfillCache struct {
	store statestore.Service

	wg sync.WaitGroup

	// backfill cache
	backfills map[string]*pb.Backfill

	// startRunRequest acts as a semaphore indicator for defining
	// when the backfillCache is being used (queried) or is it in an idle state.
	startRunRequest chan struct{}

	// requests is the channel to handle all incoming query requests and
	// make only one request to the storage service.
	requests chan *cacheRequest

	// This error is used for the return value when a request fails to reach
	// the storage. In case of multiple requests are being handled, this error
	// is returned to every single request.
	err error
}

func newBackfillCache(b *appmain.Bindings, cfg config.View) *backfillCache {
	bc := &backfillCache{
		store:           statestore.New(cfg),
		startRunRequest: make(chan struct{}, 1),
		requests:        make(chan *cacheRequest),
		backfills:       make(map[string]*pb.Backfill),
	}

	bc.startRunRequest <- struct{}{}
	b.AddHealthCheckFunc(bc.store.HealthCheck)

	return bc
}

func (bc *backfillCache) request(ctx context.Context, f func(map[string]*pb.Backfill)) error {
	cr := &cacheRequest{
		ctx:    ctx,
		runNow: make(chan struct{}),
	}

sendRequest:
	for {
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "backfill cache request canceled before request sent.")
		case <-bc.startRunRequest:
			go bc.runRequest()
		case bc.requests <- cr:
			break sendRequest
		}
	}

	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "backfill cache request canceled waiting for access.")
	case <-cr.runNow:
		defer bc.wg.Done()
	}

	if bc.err != nil {
		return bc.err
	}

	f(bc.backfills)
	return nil
}

func (bc *backfillCache) runRequest() {
	defer func() {
		bc.startRunRequest <- struct{}{}
	}()

	// Wait for first query request.
	reqs := []*cacheRequest{<-bc.requests}

	// Collect all waiting queries.
collectAllWaiting:
	for {
		select {
		case req := <-bc.requests:
			reqs = append(reqs, req)
		default:
			break collectAllWaiting
		}
	}

	bc.update()
	stats.Record(context.Background(), cacheWaitingQueries.M(int64(len(reqs))))

	// Send WaitGroup to query calls, letting them run their query on the ticket
	// cache.
	for _, req := range reqs {
		select {
		case req.runNow <- struct{}{}:
			bc.wg.Add(1)
		case <-req.ctx.Done():
		}
	}

	// wait for requests to finish using ticket cache.
	bc.wg.Wait()
}

func (bc *backfillCache) update() {
	// st := time.Now()
	previousCount := len(bc.backfills)

	currentAll, err := bc.store.GetIndexedBackfills(context.Background())
	if err != nil {
		bc.err = err
		return
	}

	deletedCount := 0
	for id := range bc.backfills {
		if _, ok := currentAll[id]; !ok {
			delete(bc.backfills, id)
			deletedCount++
		}
	}

	toFetch := []string{}
	for id := range currentAll {
		if _, ok := bc.backfills[id]; !ok {
			toFetch = append(toFetch, id)
		}
	}

	for _, backfillToFetch := range toFetch {
		bf, _, err := bc.store.GetBackfill(context.Background(), backfillToFetch)
		if err != nil {
			bc.err = err
			return
		}

		bc.backfills[bf.Id] = bf
	}

	// stats.Record(context.Background(), cacheTotalItems.M(int64(previousCount)))
	// stats.Record(context.Background(), cacheFetchedItems.M(int64(len(toFetch))))
	// stats.Record(context.Background(), cacheUpdateLatency.M(float64(time.Since(st))/float64(time.Millisecond)))

	logger.Debugf("Backfill Cache update: Previous %d, Deleted %d, Fetched %d, Current %d", previousCount, deletedCount, len(toFetch), len(bc.backfills))
	bc.err = nil
}

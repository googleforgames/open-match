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

package query

import (
	"context"
	"sync"
	"time"

	"go.opencensus.io/stats"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/appmain"
	"open-match.dev/open-match/internal/statestore"
	"open-match.dev/open-match/pkg/pb"
)

// cache unifies concurrent requests into a single cache update, and
// gives a safe view into that map cache.
type cache struct {
	store    statestore.Service
	requests chan *cacheRequest
	// Single item buffered channel.  Holds a value when runQuery can be safely
	// started.  Basically a channel/select friendly mutex around runQuery
	// running.
	startRunRequest chan struct{}
	wg              sync.WaitGroup
	// Multithreaded unsafe fields, only to be written by update, and read when
	// request given the ok.
	value  interface{}
	update func(statestore.Service, interface{}) error
	err    error
}

type cacheRequest struct {
	ctx    context.Context
	runNow chan struct{}
}

func (c *cache) request(ctx context.Context, f func(interface{})) error {
	cr := &cacheRequest{
		ctx:    ctx,
		runNow: make(chan struct{}),
	}

sendRequest:
	for {
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "cache request canceled before request sent.")
		case <-c.startRunRequest:
			go c.runRequest()
		case c.requests <- cr:
			break sendRequest
		}
	}

	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "cache request canceled waiting for access.")
	case <-cr.runNow:
		defer c.wg.Done()
	}

	if c.err != nil {
		return c.err
	}

	f(c.value)
	return nil
}

func (c *cache) runRequest() {
	defer func() {
		c.startRunRequest <- struct{}{}
	}()

	// Wait for first query request.
	reqs := []*cacheRequest{<-c.requests}

	// Collect all waiting queries.
collectAllWaiting:
	for {
		select {
		case req := <-c.requests:
			reqs = append(reqs, req)
		default:
			break collectAllWaiting
		}
	}

	c.err = c.update(c.store, c.value)
	stats.Record(context.Background(), cacheWaitingQueries.M(int64(len(reqs))))

	// Send WaitGroup to query calls, letting them run their query on the cache.
	for _, req := range reqs {
		c.wg.Add(1)
		select {
		case req.runNow <- struct{}{}:
		case <-req.ctx.Done():
			c.wg.Done()
		}
	}

	// wait for requests to finish using cache.
	c.wg.Wait()
}

func newTicketCache(b *appmain.Bindings, store statestore.Service) *cache {
	c := &cache{
		store:           store,
		requests:        make(chan *cacheRequest),
		startRunRequest: make(chan struct{}, 1),
		value:           make(map[string]*pb.Ticket),
		update:          updateTicketCache,
	}

	c.startRunRequest <- struct{}{}
	b.AddHealthCheckFunc(c.store.HealthCheck)

	return c
}

func updateTicketCache(store statestore.Service, value interface{}) error {
	if value == nil {
		return status.Error(codes.InvalidArgument, "value is required")
	}

	tickets, ok := value.(map[string]*pb.Ticket)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "expecting value type map[string]*pb.Ticket, but got: %T", value)
	}

	t := time.Now()
	previousCount := len(tickets)
	currentAll, err := store.GetIndexedIDSet(context.Background())
	if err != nil {
		return err
	}

	deletedCount := 0
	for id := range tickets {
		if _, ok := currentAll[id]; !ok {
			delete(tickets, id)
			deletedCount++
		}
	}

	toFetch := []string{}
	for id := range currentAll {
		if _, ok := tickets[id]; !ok {
			toFetch = append(toFetch, id)
		}
	}

	newTickets, err := store.GetTickets(context.Background(), toFetch)
	if err != nil {
		return err
	}

	for _, t := range newTickets {
		tickets[t.Id] = t
	}

	stats.Record(context.Background(), cacheTotalItems.M(int64(previousCount)))
	stats.Record(context.Background(), totalActiveTickets.M(int64(len(currentAll))))
	stats.Record(context.Background(), cacheFetchedItems.M(int64(len(toFetch))))
	stats.Record(context.Background(), cacheUpdateLatency.M(float64(time.Since(t))/float64(time.Millisecond)))
	stats.Record(context.Background(), totalPendingTickets.M(int64(len(toFetch))))

	logger.Debugf("Ticket Cache update: Previous %d, Deleted %d, Fetched %d, Current %d", previousCount, deletedCount, len(toFetch), len(tickets))
	return nil
}

func newBackfillCache(b *appmain.Bindings, store statestore.Service) *cache {
	c := &cache{
		store:           store,
		requests:        make(chan *cacheRequest),
		startRunRequest: make(chan struct{}, 1),
		value:           make(map[string]*pb.Backfill),
		update:          updateBackfillCache,
	}

	c.startRunRequest <- struct{}{}
	b.AddHealthCheckFunc(c.store.HealthCheck)

	return c
}

func updateBackfillCache(store statestore.Service, value interface{}) error {
	if value == nil {
		return status.Error(codes.InvalidArgument, "value is required")
	}

	backfills, ok := value.(map[string]*pb.Backfill)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "expecting value type map[string]*pb.Backfill, but got: %T", value)
	}

	t := time.Now()
	previousCount := len(backfills)
	index, err := store.GetIndexedBackfills(context.Background())
	if err != nil {
		return err
	}

	deletedCount := 0
	for id, backfill := range backfills {
		generation, ok := index[id]
		if !ok || backfill.Generation < int64(generation) {
			delete(backfills, id)
			deletedCount++
		}
	}

	toFetch := []string{}
	for id := range index {
		if _, ok := backfills[id]; !ok {
			toFetch = append(toFetch, id)
		}
	}

	fetchedBackfills, err := store.GetBackfills(context.Background(), toFetch)
	if err != nil {
		return err
	}

	for _, b := range fetchedBackfills {
		backfills[b.Id] = b
	}

	stats.Record(context.Background(), cacheTotalItems.M(int64(previousCount)))
	stats.Record(context.Background(), totalBackfillsTickets.M(int64(len(backfills))))
	stats.Record(context.Background(), cacheFetchedItems.M(int64(len(toFetch))))
	stats.Record(context.Background(), cacheUpdateLatency.M(float64(time.Since(t))/float64(time.Millisecond)))

	logger.Debugf("Backfill Cache update: Previous %d, Deleted %d, Fetched %d, Current %d", previousCount, deletedCount, len(toFetch), len(backfills))
	return nil
}

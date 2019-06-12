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

package statestore

import (
	"context"
	"fmt"

	"github.com/cenkalti/backoff"
	"github.com/gogo/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/filter"
	"open-match.dev/open-match/internal/pb"
)

var (
	redisLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "statestore.redis",
	})
)

type redisBackend struct {
	healthCheckPool *redis.Pool
	redisPool       *redis.Pool
	cfg             config.View
	ctx             context.Context
	cancelFunc      context.CancelFunc
	indexCache      chan chan *cacheResponse
}

type cacheResponse struct {
	tickets []*pb.Ticket
	err     error
}

func (rb *redisBackend) Close() error {
	rb.cancelFunc()
	return rb.redisPool.Close()
}

// newRedis creates a statestore.Service backed by Redis database.
func newRedis(cfg config.View) Service {
	// As per https://www.iana.org/assignments/uri-schemes/prov/redis
	// redis://user:secret@localhost:6379/0?foo=bar&qux=baz

	// Add redis user and password to connection url if they exist
	redisURL := "redis://"
	maskedURL := redisURL
	if cfg.IsSet("redis.password") && cfg.GetString("redis.password") != "" {
		redisURL += cfg.GetString("redis.user") + ":" + cfg.GetString("redis.password") + "@"
		maskedURL += cfg.GetString("redis.user") + ":******@"
	}
	redisURL += cfg.GetString("redis.hostname") + ":" + cfg.GetString("redis.port")
	maskedURL += cfg.GetString("redis.hostname") + ":" + cfg.GetString("redis.port")

	redisLogger.WithField("redisURL", maskedURL).Debug("Attempting to connect to Redis")

	pool := &redis.Pool{
		MaxIdle:     cfg.GetInt("redis.pool.maxIdle"),
		MaxActive:   cfg.GetInt("redis.pool.maxActive"),
		IdleTimeout: cfg.GetDuration("redis.pool.idleTimeout"),
		Dial:        func() (redis.Conn, error) { return redis.DialURL(redisURL) },
	}
	healthCheckPool := &redis.Pool{
		MaxIdle:     1,
		MaxActive:   2,
		IdleTimeout: cfg.GetDuration("redis.pool.healthCheckTimeout"),
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(redisURL, redis.DialConnectTimeout(cfg.GetDuration("redis.pool.healthCheckTimeout")), redis.DialReadTimeout(cfg.GetDuration("redis.pool.healthCheckTimeout")))
		},
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	rb := &redisBackend{
		healthCheckPool: healthCheckPool,
		redisPool:       pool,
		cfg:             cfg,
		ctx:             ctx,
		cancelFunc:      cancelFunc,
		indexCache:      make(chan chan *cacheResponse),
	}

	go rb.startCache()
	return rb
}

// HealthCheck indicates if the database is reachable.
func (rb *redisBackend) HealthCheck(ctx context.Context) error {
	redisConn, err := rb.healthCheckPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "%v", err)
	}
	defer handleConnectionClose(&redisConn)
	_, err = redisConn.Do("SELECT", "0")
	// Encountered an issue getting a connection from the pool.
	if err != nil {
		return status.Errorf(codes.Unavailable, "%v", err)
	}
	return nil
}

func (rb *redisBackend) connect(ctx context.Context) (redis.Conn, error) {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("failed to connect to redis")
		return nil, status.Errorf(codes.Unavailable, "%v", err)
	}

	return redisConn, nil
}

// CreateTicket creates a new Ticket in the state storage. If the id already exists, it will be overwritten.
func (rb *redisBackend) CreateTicket(ctx context.Context, ticket *pb.Ticket) error {
	redisConn, err := rb.connect(ctx)
	if err != nil {
		return err
	}
	defer handleConnectionClose(&redisConn)

	err = redisConn.Send("MULTI")
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"cmd":   "MULTI",
			"error": err.Error(),
		}).Error("state storage operation failed")
		return status.Errorf(codes.Internal, "%v", err)
	}

	value, err := proto.Marshal(ticket)
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"key":   ticket.Id,
			"error": err.Error(),
		}).Error("failed to marshal the ticket proto")
		return status.Errorf(codes.Internal, "%v", err)
	}

	err = redisConn.Send("SET", ticket.Id, value)
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"cmd":   "SET",
			"key":   ticket.Id,
			"error": err.Error(),
		}).Error("failed to set the value for ticket")
		return status.Errorf(codes.Internal, "%v", err)
	}

	if rb.cfg.IsSet("redis.expiration") {
		redisTTL := rb.cfg.GetInt("redis.expiration")
		if redisTTL > 0 {
			err = redisConn.Send("EXPIRE", ticket.Id, redisTTL)
			if err != nil {
				redisLogger.WithFields(logrus.Fields{
					"cmd":   "EXPIRE",
					"key":   ticket.Id,
					"ttl":   redisTTL,
					"error": err.Error(),
				}).Error("failed to set ticket expiration in state storage")
				return status.Errorf(codes.Internal, "%v", err)
			}
		}
	}

	_, err = redisConn.Do("EXEC")
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"cmd":   "EXEC",
			"key":   ticket.Id,
			"error": err.Error(),
		}).Error("failed to create ticket in state storage")
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

// GetTicket gets the Ticket with the specified id from state storage. This method fails if the Ticket does not exist.
func (rb *redisBackend) GetTicket(ctx context.Context, id string) (*pb.Ticket, error) {
	redisConn, err := rb.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer handleConnectionClose(&redisConn)

	value, err := redis.Bytes(redisConn.Do("GET", id))
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"cmd":   "GET",
			"key":   id,
			"error": err.Error(),
		}).Error("failed to get the ticket from state storage")

		// Return NotFound if redigo did not find the ticket in storage.
		if err == redis.ErrNil {
			return nil, status.Errorf(codes.NotFound, "%v", err)
		}

		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	if value == nil {
		msg := fmt.Sprintf("Ticket id:%s not found", id)
		redisLogger.WithFields(logrus.Fields{
			"key": id,
			"cmd": "GET",
		}).Error(msg)
		return nil, status.Error(codes.NotFound, msg)
	}

	ticket := &pb.Ticket{}
	err = proto.Unmarshal(value, ticket)
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"key":   id,
			"error": err.Error(),
		}).Error("failed to unmarshal the ticket proto")
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	return ticket, nil
}

// DeleteTicket removes the Ticket with the specified id from state storage.
func (rb *redisBackend) DeleteTicket(ctx context.Context, id string) error {
	redisConn, err := rb.connect(ctx)
	if err != nil {
		return err
	}
	defer handleConnectionClose(&redisConn)

	_, err = redisConn.Do("DEL", id)
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"cmd":   "DEL",
			"key":   id,
			"error": err.Error(),
		}).Error("failed to delete the ticket from state storage")
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

// IndexTicket indexes the Ticket id for the configured index fields.
func (rb *redisBackend) IndexTicket(ctx context.Context, ticket *pb.Ticket) error {
	redisConn, err := rb.connect(ctx)
	if err != nil {
		return err
	}
	defer handleConnectionClose(&redisConn)

	err = redisConn.Send("SADD", "all-tickets", ticket.Id)
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"cmd":    "SADD all-tickets",
			"ticket": ticket.Id,
			"error":  err.Error(),
		}).Error("failed to index ticket")
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

// DeindexTicket removes the indexing for the specified Ticket. Only the indexes are removed but the Ticket continues to exist.
func (rb *redisBackend) DeindexTicket(ctx context.Context, id string) error {
	redisConn, err := rb.connect(ctx)
	if err != nil {
		return err
	}
	defer handleConnectionClose(&redisConn)

	err = redisConn.Send("SREM", "all-tickets", id)
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"cmd":    "SREM all-tickets",
			"ticket": id,
			"error":  err.Error(),
		}).Error("failed to deindex ticket")
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

// FilterTickets returns the Ticket ids and required attribute key-value pairs for the Tickets meeting the specified filtering criteria.
// map[ticket.Id]map[attributeName][attributeValue]
// {
//  "testplayer1": {"ranking" : 56, "loyalty_level": 4},
//  "testplayer2": {"ranking" : 50, "loyalty_level": 3},
// }
func (rb *redisBackend) FilterTickets(ctx context.Context, filters []*pb.Filter, pageSize int, callback func([]*pb.Ticket) error) error {
	cr := make(chan *cacheResponse, 1)

	var allTickets []*pb.Ticket

	rb.indexCache <- cr

	select {
	case <-ctx.Done():
		return ctx.Err()
	case c := <-cr:
		if c.err != nil {
			return status.Errorf(codes.Internal, "%v", c.err)
		}
		allTickets = c.tickets
	}

	filtered := filter.Filter(allTickets, filters)

	// This isn't ideal.  What we would want to do is have the cache stream tickets
	// as soon as they are available.  So all of the tickets which are already locally
	// in the cache should be immediately avaiable.  However that's complex, so for
	// v0.6, let's just fudge it a little and then refine the latencies once we know
	// what we're doing more.
	for i := 0; i < len(filtered); i += pageSize {
		pageEnd := i + pageSize
		if pageEnd > len(filtered) {
			pageEnd = len(filtered)
		}

		err := callback(filtered[i:pageEnd])
		if err != nil {
			return status.Errorf(codes.Internal, "%v", err)
		}
	}

	return nil
}

// Start cache keeps a in memory cache of all indexed tickets.  When a request
// comes in, it will start a request to get all tickets ids in the index, and then
// will request unknown tickets.  If a new request comes in while another is being
// processed, the later requests will piggyback off of the first.  To preserve strict
// ordering, once the result has been returned to all waiting callers, any future
// callers will start a new request for all ticket ids from Redis.  That way a read,
// modify, read call order will return expected (not stale) results.
func (rb *redisBackend) startCache() {
	var waiting []chan *cacheResponse

	allTickets := make(map[string]*pb.Ticket)
	result := make(chan *cacheResponse, 1)

	for {
		select {
		case <-rb.ctx.Done():
			resp := &cacheResponse{
				err: rb.ctx.Err(),
			}
			for _, cr := range waiting {
				cr <- resp
			}
			return
		case resp := <-result:
			for _, cr := range waiting {
				cr <- resp
			}
			waiting = nil
		case req := <-rb.indexCache:
			if len(waiting) <= 0 {
				go rb.refillCache(allTickets, result)
			}
			waiting = append(waiting, req)
		}
	}
}

func (rb *redisBackend) refillCache(allTickets map[string]*pb.Ticket, result chan *cacheResponse) {
	redisConn, err := rb.connect(rb.ctx)
	if err != nil {
		result <- &cacheResponse{
			err: status.Errorf(codes.Internal, "%v", err),
		}
	}
	defer handleConnectionClose(&redisConn)

	allIds, err := redis.Strings(redisConn.Do("SMEMBERS", "all-tickets"))
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"Command": "SMEMBERS all-tickets",
		}).WithError(err).Error("Failed to lookup all ticket ids")
		result <- &cacheResponse{
			err: status.Errorf(codes.Internal, "%v", err),
		}
	}

	idsToRequest := make(map[string]struct{})
	for _, id := range allIds {
		idsToRequest[id] = struct{}{}
	}

	for id := range allTickets {
		if _, ok := idsToRequest[id]; ok {
			// Don't request known tickets.
			delete(idsToRequest, id)
		} else {
			// Forget deindexed tickets.
			delete(allTickets, id)
		}
	}

	req := make([]interface{}, 0, len(idsToRequest))
	for id := range idsToRequest {
		req = append(req, id)
	}

	ticketBytes, err := redis.ByteSlices(redisConn.Do("MGET", req...))
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"Command": fmt.Sprintf("MGET %v", req),
		}).WithError(err).Error("Failed to lookup tickets.")
		result <- &cacheResponse{
			err: status.Errorf(codes.Internal, "%v", err),
		}
	}

	for i, b := range ticketBytes {
		// Tickets may be deleted by the time we read it from redis.
		if b != nil {
			t := &pb.Ticket{}
			err = proto.Unmarshal(b, t)
			if err != nil {
				redisLogger.WithFields(logrus.Fields{
					"key": req[i],
				}).WithError(err).Error("Failed to unmarshal ticket from redis.")
				result <- &cacheResponse{
					err: status.Errorf(codes.Internal, "%v", err),
				}
			}
			allTickets[t.Id] = t
		}
	}

	tickets := make([]*pb.Ticket, 0, len(allTickets))
	for _, t := range allTickets {
		tickets = append(tickets, t)
	}

	result <- &cacheResponse{
		tickets: tickets,
	}
}

// UpdateAssignments update the match assignments for the input ticket ids
func (rb *redisBackend) UpdateAssignments(ctx context.Context, ids []string, assignment *pb.Assignment) error {
	if assignment == nil {
		return status.Error(codes.InvalidArgument, "assignment is nil")
	}

	redisConn, err := rb.connect(ctx)
	if err != nil {
		return err
	}
	defer handleConnectionClose(&redisConn)

	err = redisConn.Send("MULTI")
	if err != nil {
		return err
	}

	// Redis does not support roll backs See https://redis.io/topics/transactions
	// TODO: fake the batch update rollback behavior
	var ticket *pb.Ticket
	for _, id := range ids {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			ticket, err = rb.GetTicket(ctx, id)
			if err != nil {
				redisLogger.WithError(err).Errorf("failed to get ticket %s from redis when updating assignments", id)
				return err
			}

			assignmentCopy, ok := proto.Clone(assignment).(*pb.Assignment)
			if !ok {
				redisLogger.Error("failed to cast assignment object")
				return status.Error(codes.Internal, "failed to cast to the assignment object")
			}
			ticket.Assignment = assignmentCopy

			err = rb.CreateTicket(ctx, ticket)
			if err != nil {
				redisLogger.WithError(err).Errorf("failed to recreate ticket %#v with new assignment when updating assignments", ticket)
				return err
			}
		}
	}

	// Run pipelined Redis commands.
	_, err = redisConn.Do("EXEC")
	if err != nil {
		redisLogger.WithError(err).Error("failed to execute update assignments transaction")
		return err
	}

	return nil
}

// GetAssignments returns the assignment associated with the input ticket id
func (rb *redisBackend) GetAssignments(ctx context.Context, id string, callback func(*pb.Assignment) error) error {
	redisConn, err := rb.connect(ctx)
	if err != nil {
		return err
	}
	defer handleConnectionClose(&redisConn)

	backoffOperation := func() error {
		var ticket *pb.Ticket
		ticket, err = rb.GetTicket(ctx, id)
		if err != nil {
			redisLogger.WithError(err).Errorf("failed to get ticket %s when executing get assignments", id)
			return backoff.Permanent(err)
		}

		err = callback(ticket.Assignment)
		if err != nil {
			return backoff.Permanent(err)
		}

		return status.Error(codes.Unavailable, "listening on assignment updates, waiting for the next backoff")
	}

	err = backoff.Retry(backoffOperation, rb.newConstantBackoffStrategy())
	if err != nil {
		return err
	}
	return nil
}

func idsToPages(ids []string, pageSize int) [][]interface{} {
	result := make([][]interface{}, 0, len(ids)/pageSize+1)
	for i := 0; i < len(ids); i += pageSize {
		end := i + pageSize
		if end > len(ids) {
			end = len(ids)
		}
		page := make([]interface{}, end-i)
		for i, id := range ids[i:end] {
			page[i] = id
		}
		result = append(result, page)
	}
	return result
}

func handleConnectionClose(conn *redis.Conn) {
	err := (*conn).Close()
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"error": err,
		}).Debug("failed to close redis client connection.")
	}
}

func (rb *redisBackend) newConstantBackoffStrategy() backoff.BackOff {
	backoffStrat := backoff.NewConstantBackOff(rb.cfg.GetDuration("backoff.initialInterval"))
	return backoff.BackOff(backoffStrat)
}

// TODO: add cache the backoff object
// nolint: unused
func (rb *redisBackend) newExponentialBackoffStrategy() backoff.BackOff {
	backoffStrat := backoff.NewExponentialBackOff()
	backoffStrat.InitialInterval = rb.cfg.GetDuration("backoff.initialInterval")
	backoffStrat.RandomizationFactor = rb.cfg.GetFloat64("backoff.randFactor")
	backoffStrat.Multiplier = rb.cfg.GetFloat64("backoff.multiplier")
	backoffStrat.MaxInterval = rb.cfg.GetDuration("backoff.maxInterval")
	backoffStrat.MaxElapsedTime = rb.cfg.GetDuration("backoff.maxElapsedTime")
	return backoff.BackOff(backoffStrat)
}

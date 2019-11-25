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
	"io/ioutil"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/set"
	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

var (
	redisLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "statestore.redis",
	})
	mRedisConnLatencyMs  = telemetry.HistogramWithBounds("redis/connectlatency", "latency to get a redis connection", "ms", telemetry.HistogramBounds)
	mRedisConnPoolActive = telemetry.Gauge("redis/connectactivecount", "number of connections in the pool, includes idle plus connections in use")
	mRedisConnPoolIdle   = telemetry.Gauge("redis/connectidlecount", "number of idle connections in the pool")
)

type redisBackend struct {
	healthCheckPool *redis.Pool
	redisPool       *redis.Pool
	cfg             config.View
}

// Close the connection to the database.
func (rb *redisBackend) Close() error {
	return rb.redisPool.Close()
}

// newRedis creates a statestore.Service backed by Redis database.
func newRedis(cfg config.View) Service {
	// As per https://www.iana.org/assignments/uri-schemes/prov/redis
	// redis://user:secret@localhost:6379/0?foo=bar&qux=baz

	// Add redis user and password to connection url if they exist
	redisURL := "redis://"
	maskedURL := redisURL

	passwordFile := cfg.GetString("redis.passwordPath")
	if len(passwordFile) > 0 {
		redisLogger.Debugf("loading Redis password from file %s", passwordFile)
		passwordData, err := ioutil.ReadFile(passwordFile)
		if err != nil {
			redisLogger.Fatalf("cannot read Redis password from file %s, desc: %s", passwordFile, err.Error())
		}
		redisURL += fmt.Sprintf("%s:%s@", cfg.GetString("redis.user"), string(passwordData))
		maskedURL += fmt.Sprintf("%s:%s@", cfg.GetString("redis.user"), "**********")
	}
	redisURL += cfg.GetString("redis.hostname") + ":" + cfg.GetString("redis.port")
	maskedURL += cfg.GetString("redis.hostname") + ":" + cfg.GetString("redis.port")

	redisLogger.WithField("redisURL", maskedURL).Debug("Attempting to connect to Redis")

	pool := &redis.Pool{
		MaxIdle:     cfg.GetInt("redis.pool.maxIdle"),
		MaxActive:   cfg.GetInt("redis.pool.maxActive"),
		IdleTimeout: cfg.GetDuration("redis.pool.idleTimeout"),
		Wait:        true,
		TestOnBorrow: func(c redis.Conn, _ time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		DialContext: func(ctx context.Context) (redis.Conn, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return redis.DialURL(redisURL, redis.DialConnectTimeout(cfg.GetDuration("redis.pool.idleTimeout")), redis.DialReadTimeout(cfg.GetDuration("redis.pool.idleTimeout")))
		},
	}
	healthCheckPool := &redis.Pool{
		MaxIdle:     1,
		MaxActive:   2,
		IdleTimeout: cfg.GetDuration("redis.pool.healthCheckTimeout"),
		Wait:        true,
		DialContext: func(ctx context.Context) (redis.Conn, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return redis.DialURL(redisURL, redis.DialConnectTimeout(cfg.GetDuration("redis.pool.healthCheckTimeout")), redis.DialReadTimeout(cfg.GetDuration("redis.pool.healthCheckTimeout")))
		},
	}

	return &redisBackend{
		healthCheckPool: healthCheckPool,
		redisPool:       pool,
		cfg:             cfg,
	}
}

// HealthCheck indicates if the database is reachable.
func (rb *redisBackend) HealthCheck(ctx context.Context) error {
	redisConn, err := rb.healthCheckPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "%v", err)
	}
	defer handleConnectionClose(&redisConn)

	poolStats := rb.redisPool.Stats()
	telemetry.SetGauge(ctx, mRedisConnPoolActive, int64(poolStats.ActiveCount))
	telemetry.SetGauge(ctx, mRedisConnPoolIdle, int64(poolStats.IdleCount))

	_, err = redisConn.Do("PING")
	// Encountered an issue getting a connection from the pool.
	if err != nil {
		return status.Errorf(codes.Unavailable, "%v", err)
	}

	return nil
}

func (rb *redisBackend) connect(ctx context.Context) (redis.Conn, error) {
	startTime := time.Now()
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("failed to connect to redis")
		return nil, status.Errorf(codes.Unavailable, "%v", err)
	}
	telemetry.RecordNUnitMeasurement(ctx, mRedisConnLatencyMs, time.Since(startTime).Milliseconds())

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
			"key":   ticket.GetId(),
			"error": err.Error(),
		}).Error("failed to marshal the ticket proto")
		return status.Errorf(codes.Internal, "%v", err)
	}

	err = redisConn.Send("SET", ticket.GetId(), value)
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"cmd":   "SET",
			"key":   ticket.GetId(),
			"error": err.Error(),
		}).Error("failed to set the value for ticket")
		return status.Errorf(codes.Internal, "%v", err)
	}

	if rb.cfg.IsSet("redis.expiration") {
		redisTTL := rb.cfg.GetInt("redis.expiration")
		if redisTTL > 0 {
			err = redisConn.Send("EXPIRE", ticket.GetId(), redisTTL)
			if err != nil {
				redisLogger.WithFields(logrus.Fields{
					"cmd":   "EXPIRE",
					"key":   ticket.GetId(),
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
			"key":   ticket.GetId(),
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
			msg := fmt.Sprintf("Ticket id:%s not found", id)
			redisLogger.WithFields(logrus.Fields{
				"key": id,
				"cmd": "GET",
			}).Error(msg)
			return nil, status.Error(codes.NotFound, msg)
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

	indexedFields := extractIndexedFields(ticket)

	err = redisConn.Send("MULTI")
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"cmd":   "MULTI",
			"error": err.Error(),
		}).Error("state storage operation failed")
		return status.Errorf(codes.Internal, "%v", err)
	}

	{
		command := make([]interface{}, 0, 1+len(indexedFields))
		command = append(command, indexCacheName(ticket.Id))
		for index := range indexedFields {
			command = append(command, index)
		}
		err = redisConn.Send("SADD", command...)
		if err != nil {
			redisLogger.WithFields(logrus.Fields{
				"cmd":     "SADD",
				"ticket":  ticket.GetId(),
				"error":   err.Error(),
				"indices": command[1:],
			}).Error("failed to set ticket's indices")
			return status.Errorf(codes.Internal, "%v", err)
		}
	}

	for k, v := range indexedFields {
		// Index the DoubleArg by value.
		err = redisConn.Send("ZADD", k, v, ticket.GetId())
		if err != nil {
			redisLogger.WithFields(logrus.Fields{
				"cmd":       "ZADD",
				"DoubleArg": k,
				"value":     v,
				"ticket":    ticket.GetId(),
				"error":     err.Error(),
			}).Error("failed to index ticket DoubleArg")
			return status.Errorf(codes.Internal, "%v", err)
		}
	}

	// Run pipelined Redis commands.
	_, err = redisConn.Do("EXEC")
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"cmd":   "EXEC",
			"id":    ticket.GetId(),
			"error": err.Error(),
		}).Error("failed to index the ticket")
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

	indices, err := redis.Strings(redisConn.Do("SMEMBERS", indexCacheName(id)))
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"SMEMBERS": "MULTI",
			"error":    err.Error(),
			"ticket":   id,
		}).Error("failed to retrieve ticket's indexed fields")
		return status.Errorf(codes.Internal, "%v", err)
	}

	if len(indices) == 0 {
		return nil
	}

	err = redisConn.Send("MULTI")
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"cmd":   "MULTI",
			"error": err.Error(),
		}).Error("state storage operation failed")
		return status.Errorf(codes.Internal, "%v", err)
	}

	for _, index := range indices {
		err = redisConn.Send("ZREM", index, id)
		if err != nil {
			redisLogger.WithFields(logrus.Fields{
				"cmd":   "ZREM",
				"index": index,
				"id":    id,
				"error": err.Error(),
			}).Error("failed to deindex the ticket")
			return status.Errorf(codes.Internal, "%v", err)
		}
	}

	err = redisConn.Send("DEL", indexCacheName(id))
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"cmd":   "DEL",
			"id":    id,
			"error": err.Error(),
		}).Error("failed to remove ticket's indexed fields")
		return status.Errorf(codes.Internal, "%v", err)
	}

	_, err = redisConn.Do("EXEC")
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"cmd":   "EXEC",
			"id":    id,
			"error": err.Error(),
		}).Error("failed to deindex the ticket")
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

// FilterTickets returns the Ticket ids and required DoubleArg key-value pairs for the Tickets meeting the specified filtering criteria.
// map[ticket.Id]map[DoubleArgName][DoubleArgValue]
// {
//  "testplayer1": {"ranking" : 56, "loyalty_level": 4},
//  "testplayer2": {"ranking" : 50, "loyalty_level": 3},
// }
func (rb *redisBackend) FilterTickets(ctx context.Context, pool *pb.Pool, pageSize int, callback func([]*pb.Ticket) error) error {
	var err error
	var redisConn redis.Conn
	var ticketBytes [][]byte
	var idsInFilter, idsInIgnoreLists []string

	redisConn, err = rb.connect(ctx)
	if err != nil {
		return err
	}
	defer handleConnectionClose(&redisConn)

	ttl := rb.cfg.GetDuration("redis.ignoreLists.ttl")
	curTime := time.Now()
	curTimeInt := curTime.UnixNano()
	startTimeInt := curTime.Add(-ttl).UnixNano()

	// Filter out tickets that are fetched but not assigned within ttl time (ms).
	idsInIgnoreLists, err = redis.Strings(redisConn.Do("ZRANGEBYSCORE", "proposed_ticket_ids", startTimeInt, curTimeInt))
	if err != nil {
		redisLogger.WithError(err).Error("failed to get proposed tickets")
		return status.Errorf(codes.Internal, err.Error())
	}

	// A set of playerIds that satisfies all filters
	idSet := make([]string, 0)

	// For each filter, do a range query to Redis on Filter.DoubleArg
	for i, filter := range extractIndexFilters(pool) {
		// Time Complexity O(logN + M), where N is the number of elements in the DoubleArg set
		// and M is the number of entries being returned.
		// TODO: discuss if we need a LIMIT for # of queries being returned
		idsInFilter, err = redis.Strings(redisConn.Do("ZRANGEBYSCORE", filter.name, filter.min, filter.max))
		if err != nil {
			redisLogger.WithFields(logrus.Fields{
				"Command": fmt.Sprintf("ZRANGEBYSCORE %s %f %f", filter.name, filter.min, filter.max),
			}).WithError(err).Error("Failed to lookup index.")
			return status.Errorf(codes.Internal, "%v", err)
		}

		if i == 0 {
			idSet = idsInFilter
		} else {
			idSet = set.Intersection(idSet, idsInFilter)
		}
	}

	idSet = set.Difference(idSet, idsInIgnoreLists)

	// TODO: finish reworking this after the proto changes.
	for _, page := range idsToPages(idSet, pageSize) {
		ticketBytes, err = redis.ByteSlices(redisConn.Do("MGET", page...))
		if err != nil {
			redisLogger.WithFields(logrus.Fields{
				"Command": fmt.Sprintf("MGET %v", page),
			}).WithError(err).Error("Failed to lookup tickets.")
			return status.Errorf(codes.Internal, "%v", err)
		}

		tickets := make([]*pb.Ticket, 0, len(page))
		for i, b := range ticketBytes {
			// Tickets may be deleted by the time we read it from redis.
			if b != nil {
				t := &pb.Ticket{}
				err = proto.Unmarshal(b, t)
				if err != nil {
					redisLogger.WithFields(logrus.Fields{
						"key": page[i],
					}).WithError(err).Error("Failed to unmarshal ticket from redis.")
					return status.Errorf(codes.Internal, "%v", err)
				}
				tickets = append(tickets, t)
			}
		}

		err = callback(tickets)
		if err != nil {
			return status.Errorf(codes.Internal, "%v", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}

// UpdateAssignments update the match assignments for the input ticket ids.
// This function guarantees if any of the input ids does not exists, the state of the storage service won't be altered.
// However, since Redis does not support transaction roll backs (see https://redis.io/topics/transactions), some of the
// assignment fields might be partially updated if this function encounters an error halfway through the execution.
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

	// Sanity check to make sure all inputs ids are valid
	tickets := []*pb.Ticket{}
	for _, id := range ids {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var ticket *pb.Ticket
			ticket, err = rb.GetTicket(ctx, id)
			if err != nil {
				redisLogger.WithError(err).Errorf("failed to get ticket %s from redis when updating assignments", id)
				return err
			}
			tickets = append(tickets, ticket)
		}
	}

	for _, ticket := range tickets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
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

		err = callback(ticket.GetAssignment())
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

// AddProposedTickets appends new proposed tickets to the proposed sorted set with current timestamp
func (rb *redisBackend) AddTicketsToIgnoreList(ctx context.Context, ids []string) error {
	redisConn, err := rb.connect(ctx)
	if err != nil {
		return err
	}
	defer handleConnectionClose(&redisConn)

	err = redisConn.Send("MULTI")
	if err != nil {
		redisLogger.WithError(err).Error("failed to pipeline commands for AddTicketsToIgnoreList")
		return status.Error(codes.Internal, err.Error())
	}

	currentTime := time.Now().UnixNano()
	for _, id := range ids {
		// Index the DoubleArg by value.
		err = redisConn.Send("ZADD", "proposed_ticket_ids", currentTime, id)
		if err != nil {
			redisLogger.WithError(err).Error("failed to append proposed tickets to redis")
			return status.Error(codes.Internal, err.Error())
		}
	}

	// Run pipelined Redis commands.
	_, err = redisConn.Do("EXEC")
	if err != nil {
		redisLogger.WithError(err).Error("failed to execute pipelined commands for AddTicketsToIgnoreList")
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}

// DeleteTicketsFromIgnoreList deletes tickets from the proposed sorted set
func (rb *redisBackend) DeleteTicketsFromIgnoreList(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	redisConn, err := rb.connect(ctx)
	if err != nil {
		return err
	}
	defer handleConnectionClose(&redisConn)

	err = redisConn.Send("MULTI")
	if err != nil {
		redisLogger.WithError(err).Error("failed to pipeline commands for DeleteTicketsFromIgnoreList")
		return status.Error(codes.Internal, err.Error())
	}

	for _, id := range ids {
		err = redisConn.Send("ZREM", "proposed_ticket_ids", id)
		if err != nil {
			redisLogger.WithError(err).Error("failed to delete proposed tickets from ignore list")
			return status.Error(codes.Internal, err.Error())
		}
	}

	// Run pipelined Redis commands.
	_, err = redisConn.Do("EXEC")
	if err != nil {
		redisLogger.WithError(err).Error("failed to execute pipelined commands for DeleteTicketsFromIgnoreList")
		return status.Error(codes.Internal, err.Error())
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

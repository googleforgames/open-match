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
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/pkg/pb"
)

const (
	allTickets        = "allTickets"
	proposedTicketIDs = "proposed_ticket_ids"
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
}

// Close the connection to the database.
func (rb *redisBackend) Close() error {
	return rb.redisPool.Close()
}

// newRedis creates a statestore.Service backed by Redis database.
func newRedis(cfg config.View) Service {
	return &redisBackend{
		healthCheckPool: getHealthCheckPool(cfg),
		redisPool:       GetRedisPool(cfg),
		cfg:             cfg,
	}
}

func getHealthCheckPool(cfg config.View) *redis.Pool {
	var healthCheckURL string
	var maxIdle = 3
	var maxActive = 0
	var healthCheckTimeout = cfg.GetDuration("redis.pool.healthCheckTimeout")

	if cfg.IsSet("redis.sentinelHostname") {
		sentinelAddr := getSentinelAddr(cfg)
		healthCheckURL = redisURLFromAddr(sentinelAddr, cfg, cfg.GetBool("redis.sentinelUsePassword"))
	} else {
		masterAddr := getMasterAddr(cfg)
		healthCheckURL = redisURLFromAddr(masterAddr, cfg, cfg.GetBool("redis.usePassword"))
	}

	return &redis.Pool{
		MaxIdle:      maxIdle,
		MaxActive:    maxActive,
		IdleTimeout:  10 * healthCheckTimeout,
		Wait:         true,
		TestOnBorrow: testOnBorrow,
		DialContext: func(ctx context.Context) (redis.Conn, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return redis.DialURL(healthCheckURL, redis.DialConnectTimeout(healthCheckTimeout), redis.DialReadTimeout(healthCheckTimeout))
		},
	}
}

// GetRedisPool configures a new pool to connect to redis given the config.
func GetRedisPool(cfg config.View) *redis.Pool {
	var dialFunc func(context.Context) (redis.Conn, error)
	maxIdle := cfg.GetInt("redis.pool.maxIdle")
	maxActive := cfg.GetInt("redis.pool.maxActive")
	idleTimeout := cfg.GetDuration("redis.pool.idleTimeout")

	if cfg.IsSet("redis.sentinelHostname") {
		sentinelPool := getSentinelPool(cfg)
		dialFunc = func(ctx context.Context) (redis.Conn, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}

			sentinelConn, err := sentinelPool.GetContext(ctx)
			if err != nil {
				redisLogger.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("failed to connect to redis sentinel")
				return nil, status.Errorf(codes.Unavailable, "%v", err)
			}

			masterInfo, err := redis.Strings(sentinelConn.Do("SENTINEL", "GET-MASTER-ADDR-BY-NAME", cfg.GetString("redis.sentinelMaster")))
			if err != nil {
				redisLogger.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("failed to get current master from redis sentinel")
				return nil, status.Errorf(codes.Unavailable, "%v", err)
			}

			masterURL := redisURLFromAddr(fmt.Sprintf("%s:%s", masterInfo[0], masterInfo[1]), cfg, cfg.GetBool("redis.usePassword"))
			return redis.DialURL(masterURL, redis.DialConnectTimeout(idleTimeout), redis.DialReadTimeout(idleTimeout))
		}
	} else {
		masterAddr := getMasterAddr(cfg)
		masterURL := redisURLFromAddr(masterAddr, cfg, cfg.GetBool("redis.usePassword"))
		dialFunc = func(ctx context.Context) (redis.Conn, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return redis.DialURL(masterURL, redis.DialConnectTimeout(idleTimeout), redis.DialReadTimeout(idleTimeout))
		}
	}

	return &redis.Pool{
		MaxIdle:      maxIdle,
		MaxActive:    maxActive,
		IdleTimeout:  idleTimeout,
		Wait:         true,
		TestOnBorrow: testOnBorrow,
		DialContext:  dialFunc,
	}
}

func getSentinelPool(cfg config.View) *redis.Pool {
	maxIdle := cfg.GetInt("redis.pool.maxIdle")
	maxActive := cfg.GetInt("redis.pool.maxActive")
	idleTimeout := cfg.GetDuration("redis.pool.idleTimeout")

	sentinelAddr := getSentinelAddr(cfg)
	sentinelURL := redisURLFromAddr(sentinelAddr, cfg, cfg.GetBool("redis.sentinelUsePassword"))
	return &redis.Pool{
		MaxIdle:      maxIdle,
		MaxActive:    maxActive,
		IdleTimeout:  idleTimeout,
		Wait:         true,
		TestOnBorrow: testOnBorrow,
		DialContext: func(ctx context.Context) (redis.Conn, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			redisLogger.WithField("sentinelAddr", sentinelAddr).Debug("Attempting to connect to Redis Sentinel")
			return redis.DialURL(sentinelURL, redis.DialConnectTimeout(idleTimeout), redis.DialReadTimeout(idleTimeout))
		},
	}
}

// HealthCheck indicates if the database is reachable.
func (rb *redisBackend) HealthCheck(ctx context.Context) error {
	redisConn, err := rb.healthCheckPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "%v", err)
	}
	defer handleConnectionClose(&redisConn)

	_, err = redisConn.Do("PING")
	// Encountered an issue getting a connection from the pool.
	if err != nil {
		return status.Errorf(codes.Unavailable, "%v", err)
	}
	return nil
}

func testOnBorrow(c redis.Conn, lastUsed time.Time) error {
	// Assume the connection is valid if it was used in 30 sec.
	if time.Since(lastUsed) < 15*time.Second {
		return nil
	}

	_, err := c.Do("PING")
	return err
}

func getSentinelAddr(cfg config.View) string {
	return fmt.Sprintf("%s:%s", cfg.GetString("redis.sentinelHostname"), cfg.GetString("redis.sentinelPort"))
}

func getMasterAddr(cfg config.View) string {
	return fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port"))
}

func redisURLFromAddr(addr string, cfg config.View, usePassword bool) string {
	// As per https://www.iana.org/assignments/uri-schemes/prov/redis
	// redis://user:secret@localhost:6379/0?foo=bar&qux=baz

	// Add redis user and password to connection url if they exist
	redisURL := "redis://"

	if usePassword {
		passwordFile := cfg.GetString("redis.passwordPath")
		redisLogger.Debugf("loading Redis password from file %s", passwordFile)
		passwordData, err := ioutil.ReadFile(passwordFile)
		if err != nil {
			redisLogger.Fatalf("cannot read Redis password from file %s, desc: %s", passwordFile, err.Error())
		}
		redisURL += fmt.Sprintf("%s:%s@", cfg.GetString("redis.user"), string(passwordData))
	}

	return redisURL + addr
}

// CreateTicket creates a new Ticket in the state storage. If the id already exists, it will be overwritten.
func (rb *redisBackend) CreateTicket(ctx context.Context, ticket *pb.Ticket) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "CreateTicket, id: %s, failed to connect to redis: %v", ticket.GetId(), err)
	}
	defer handleConnectionClose(&redisConn)

	value, err := proto.Marshal(ticket)
	if err != nil {
		err = errors.Wrapf(err, "failed to marshal the ticket proto, id: %s", ticket.GetId())
		return status.Errorf(codes.Internal, "%v", err)
	}

	_, err = redisConn.Do("SET", ticket.GetId(), value)
	if err != nil {
		err = errors.Wrapf(err, "failed to set the value for ticket, id: %s", ticket.GetId())
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

// GetTicket gets the Ticket with the specified id from state storage. This method fails if the Ticket does not exist.
func (rb *redisBackend) GetTicket(ctx context.Context, id string) (*pb.Ticket, error) {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "GetTicket, id: %s, failed to connect to redis: %v", id, err)
	}
	defer handleConnectionClose(&redisConn)

	value, err := redis.Bytes(redisConn.Do("GET", id))
	if err != nil {
		// Return NotFound if redigo did not find the ticket in storage.
		if err == redis.ErrNil {
			msg := fmt.Sprintf("Ticket id: %s not found", id)
			return nil, status.Error(codes.NotFound, msg)
		}

		err = errors.Wrapf(err, "failed to get the ticket from state storage, id: %s", id)
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	if value == nil {
		msg := fmt.Sprintf("Ticket id: %s not found", id)
		return nil, status.Error(codes.NotFound, msg)
	}

	ticket := &pb.Ticket{}
	err = proto.Unmarshal(value, ticket)
	if err != nil {
		err = errors.Wrapf(err, "failed to unmarshal the ticket proto, id: %s", id)
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	return ticket, nil
}

// DeleteTicket removes the Ticket with the specified id from state storage.
func (rb *redisBackend) DeleteTicket(ctx context.Context, id string) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "DeleteTicket, id: %s, failed to connect to redis: %v", id, err)
	}
	defer handleConnectionClose(&redisConn)

	_, err = redisConn.Do("DEL", id)
	if err != nil {
		err = errors.Wrapf(err, "failed to delete the ticket from state storage, id: %s", id)
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

// IndexTicket indexes the Ticket id for the configured index fields.
func (rb *redisBackend) IndexTicket(ctx context.Context, ticket *pb.Ticket) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "IndexTicket, id: %s, failed to connect to redis: %v", ticket.GetId(), err)
	}
	defer handleConnectionClose(&redisConn)

	err = redisConn.Send("SADD", allTickets, ticket.Id)
	if err != nil {
		err = errors.Wrapf(err, "failed to add ticket to all tickets, id: %s", ticket.Id)
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

// DeindexTicket removes the indexing for the specified Ticket. Only the indexes are removed but the Ticket continues to exist.
func (rb *redisBackend) DeindexTicket(ctx context.Context, id string) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "DeindexTicket, id: %s, failed to connect to redis: %v", id, err)
	}
	defer handleConnectionClose(&redisConn)

	err = redisConn.Send("SREM", allTickets, id)
	if err != nil {
		err = errors.Wrapf(err, "failed to remove ticket from all tickets, id: %s", id)
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

// GetIndexedIds returns the ids of all tickets currently indexed.
func (rb *redisBackend) GetIndexedIDSet(ctx context.Context) (map[string]struct{}, error) {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "GetIndexedIDSet, failed to connect to redis: %v", err)
	}
	defer handleConnectionClose(&redisConn)

	ttl := rb.cfg.GetDuration("pendingReleaseTimeout")
	curTime := time.Now()
	endTimeInt := curTime.Add(time.Hour).UnixNano()
	startTimeInt := curTime.Add(-ttl).UnixNano()

	// Filter out tickets that are fetched but not assigned within ttl time (ms).
	idsInPendingReleases, err := redis.Strings(redisConn.Do("ZRANGEBYSCORE", proposedTicketIDs, startTimeInt, endTimeInt))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting pending release %v", err)
	}

	idsIndexed, err := redis.Strings(redisConn.Do("SMEMBERS", allTickets))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting all indexed ticket ids %v", err)
	}

	r := make(map[string]struct{}, len(idsIndexed))
	for _, id := range idsIndexed {
		r[id] = struct{}{}
	}
	for _, id := range idsInPendingReleases {
		delete(r, id)
	}

	return r, nil
}

// GetTickets returns multiple tickets from storage.  Missing tickets are
// silently ignored.
func (rb *redisBackend) GetTickets(ctx context.Context, ids []string) ([]*pb.Ticket, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "GetTickets, failed to connect to redis: %v", err)
	}
	defer handleConnectionClose(&redisConn)

	queryParams := make([]interface{}, len(ids))
	for i, id := range ids {
		queryParams[i] = id
	}

	ticketBytes, err := redis.ByteSlices(redisConn.Do("MGET", queryParams...))
	if err != nil {
		err = errors.Wrapf(err, "failed to lookup tickets %v", ids)
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	r := make([]*pb.Ticket, 0, len(ids))

	for i, b := range ticketBytes {
		// Tickets may be deleted by the time we read it from redis.
		if b != nil {
			t := &pb.Ticket{}
			err = proto.Unmarshal(b, t)
			if err != nil {
				err = errors.Wrapf(err, "failed to unmarshal ticket from redis, key %s", ids[i])
				return nil, status.Errorf(codes.Internal, "%v", err)
			}
			r = append(r, t)
		}
	}

	return r, nil
}

// UpdateAssignments update using the request's specified tickets with assignments.
func (rb *redisBackend) UpdateAssignments(ctx context.Context, req *pb.AssignTicketsRequest) (*pb.AssignTicketsResponse, []*pb.Ticket, error) {
	resp := &pb.AssignTicketsResponse{}
	if len(req.Assignments) == 0 {
		return resp, []*pb.Ticket{}, nil
	}

	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return nil, nil, status.Errorf(codes.Unavailable, "UpdateAssignments, failed to connect to redis: %v", err)
	}
	defer handleConnectionClose(&redisConn)

	idToA := make(map[string]*pb.Assignment)
	ids := make([]string, 0)
	idsI := make([]interface{}, 0)
	for _, a := range req.Assignments {
		if a.Assignment == nil {
			return nil, nil, status.Error(codes.InvalidArgument, "AssignmentGroup.Assignment is required")
		}

		for _, id := range a.TicketIds {
			if _, ok := idToA[id]; ok {
				return nil, nil, status.Errorf(codes.InvalidArgument, "Ticket id %s is assigned multiple times in one assign tickets call", id)
			}

			idToA[id] = a.Assignment
			ids = append(ids, id)
			idsI = append(idsI, id)
		}
	}

	ticketBytes, err := redis.ByteSlices(redisConn.Do("MGET", idsI...))
	if err != nil {
		return nil, nil, err
	}

	tickets := make([]*pb.Ticket, 0, len(ticketBytes))
	for i, ticketByte := range ticketBytes {
		// Tickets may be deleted by the time we read it from redis.
		if ticketByte == nil {
			resp.Failures = append(resp.Failures, &pb.AssignmentFailure{
				TicketId: ids[i],
				Cause:    pb.AssignmentFailure_TICKET_NOT_FOUND,
			})
		} else {
			t := &pb.Ticket{}
			err = proto.Unmarshal(ticketByte, t)
			if err != nil {
				err = errors.Wrapf(err, "failed to unmarshal ticket from redis %s", ids[i])
				return nil, nil, status.Errorf(codes.Internal, "%v", err)
			}
			tickets = append(tickets, t)
		}
	}
	assignmentTimeout := rb.cfg.GetDuration("assignedDeleteTimeout") / time.Millisecond
	err = redisConn.Send("MULTI")
	if err != nil {
		return nil, nil, errors.Wrap(err, "error starting redis multi")
	}

	for _, ticket := range tickets {
		ticket.Assignment = idToA[ticket.Id]

		var ticketByte []byte
		ticketByte, err = proto.Marshal(ticket)
		if err != nil {
			return nil, nil, status.Errorf(codes.Internal, "failed to marshal ticket %s", ticket.GetId())
		}

		err = redisConn.Send("SET", ticket.Id, ticketByte, "PX", int64(assignmentTimeout), "XX")
		if err != nil {
			return nil, nil, errors.Wrap(err, "error sending ticket assignment set")
		}
	}

	wasSet, err := redis.Values(redisConn.Do("EXEC"))
	if err != nil {
		return nil, nil, errors.Wrap(err, "error executing assignment set")
	}

	if len(wasSet) != len(tickets) {
		return nil, nil, status.Errorf(codes.Internal, "sent %d tickets to redis, but received %d back", len(tickets), len(wasSet))
	}

	assignedTickets := make([]*pb.Ticket, 0, len(tickets))
	for i, ticket := range tickets {
		v, err := redis.String(wasSet[i], nil)
		if err == redis.ErrNil {
			resp.Failures = append(resp.Failures, &pb.AssignmentFailure{
				TicketId: ticket.Id,
				Cause:    pb.AssignmentFailure_TICKET_NOT_FOUND,
			})
			continue
		}
		if err != nil {
			return nil, nil, errors.Wrap(err, "unexpected error from redis multi set")
		}
		if v != "OK" {
			return nil, nil, status.Errorf(codes.Internal, "unexpected response from redis: %s", v)
		}
		assignedTickets = append(assignedTickets, ticket)
	}

	return resp, assignedTickets, nil
}

// GetAssignments returns the assignment associated with the input ticket id
func (rb *redisBackend) GetAssignments(ctx context.Context, id string, callback func(*pb.Assignment) error) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "GetAssignments, id: %s, failed to connect to redis: %v", id, err)
	}
	defer handleConnectionClose(&redisConn)

	backoffOperation := func() error {
		var ticket *pb.Ticket
		ticket, err = rb.GetTicket(ctx, id)
		if err != nil {
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

// AddTicketsToPendingRelease appends new proposed tickets to the proposed sorted set with current timestamp
func (rb *redisBackend) AddTicketsToPendingRelease(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "AddTicketsToPendingRelease, failed to connect to redis: %v", err)
	}
	defer handleConnectionClose(&redisConn)

	currentTime := time.Now().UnixNano()
	cmds := make([]interface{}, 0, 2*len(ids)+1)
	cmds = append(cmds, proposedTicketIDs)
	for _, id := range ids {
		cmds = append(cmds, currentTime, id)
	}

	_, err = redisConn.Do("ZADD", cmds...)
	if err != nil {
		err = errors.Wrap(err, "failed to append proposed tickets to pending release")
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}

// DeleteTicketsFromPendingRelease deletes tickets from the proposed sorted set
func (rb *redisBackend) DeleteTicketsFromPendingRelease(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "DeleteTicketsFromPendingRelease, failed to connect to redis: %v", err)
	}
	defer handleConnectionClose(&redisConn)

	cmds := make([]interface{}, 0, len(ids)+1)
	cmds = append(cmds, proposedTicketIDs)
	for _, id := range ids {
		cmds = append(cmds, id)
	}

	_, err = redisConn.Do("ZREM", cmds...)
	if err != nil {
		err = errors.Wrap(err, "failed to delete proposed tickets from pending release")
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}

func (rb *redisBackend) ReleaseAllTickets(ctx context.Context) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "ReleaseAllTickets, failed to connect to redis: %v", err)
	}
	defer handleConnectionClose(&redisConn)

	_, err = redisConn.Do("DEL", proposedTicketIDs)
	return err
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

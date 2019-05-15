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
	"math"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/set"
)

var (
	redisLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "statestore.redis",
	})
)

type redisBackend struct {
	redisPool *redis.Pool
	cfg       config.View
}

func (rb *redisBackend) Close() error {
	return rb.redisPool.Close()
}

// newRedis creates a statestore.Service backed by Redis database.
func newRedis(cfg config.View) (Service, error) {
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
		IdleTimeout: cfg.GetDuration("redis.pool.idleTimeout") * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.DialURL(redisURL) },
	}

	// Sanity check that connection works before passing it back.  Redigo
	// always returns a valid connection, and will just fail on the first
	// query: https://godoc.org/github.com/gomodule/redigo/redis#Pool.Get
	redisConn := pool.Get()
	defer handleConnectionClose(&redisConn)

	_, err := redisConn.Do("SELECT", "0")
	// Encountered an issue getting a connection from the pool.
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"cmd":   "SELECT 0",
			"error": err.Error()}).Error("state storage connection error")
		return nil, fmt.Errorf("cannot connect to Redis at %s, %s", maskedURL, err)
	}

	redisLogger.Info("Connected to Redis")
	return &redisBackend{
		redisPool: pool,
		cfg:       cfg,
	}, nil
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

// IndexTicket indexes the Ticket id for the specified fields.
func (rb *redisBackend) IndexTicket(ctx context.Context, ticket pb.Ticket, indices []string) error {
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

	// Loop through all attributes we want to index.
	for _, attribute := range indices {
		// TODO: Currently, Open Match supports specifying attributes as JSON properties on Tickets.
		// Alternatives for populating this information in proto fields are being considered.
		// Also, we need to add Ticket creation time to either the ticket or index.
		// Meta characters bein specified in JSON property keys is currently not supported.
		v := gjson.Get(ticket.Properties, attribute)

		// If this attribute wasn't provided in the JSON, continue to the next attribute to index.
		if !v.Exists() {
			redisLogger.WithFields(logrus.Fields{
				"attribute": attribute}).Warning("Couldn't find index in Ticket Properties")
			continue
		}

		// Value exists. Check if it is a supported value for indexed properties.
		if v.Int() < math.MaxInt64 || v.Int() > math.MaxInt64 {
			redisLogger.WithFields(logrus.Fields{
				"attribute": attribute}).Warning("Invalid value for attribute, skip indexing.")
			continue
		}

		// Index the attribute by value.
		value := v.Int()
		err = redisConn.Send("ZADD", attribute, value, ticket.Id)
		if err != nil {
			redisLogger.WithFields(logrus.Fields{
				"cmd":       "ZADD",
				"attribute": attribute,
				"value":     value,
				"ticket":    ticket.Id,
				"error":     err.Error(),
			}).Error("failed to index ticket attribute")
			return status.Errorf(codes.Internal, "%v", err)
		}
	}

	// Run pipelined Redis commands.
	_, err = redisConn.Do("EXEC")
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"cmd":   "EXEC",
			"id":    ticket.Id,
			"error": err.Error(),
		}).Error("failed to index the ticket")
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

// DeindexTicket removes the indexing for the specified Ticket. Only the indexes are removed but the Ticket continues to exist.
func (rb *redisBackend) DeindexTicket(ctx context.Context, id string, indices []string) error {
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

	for _, attribute := range indices {
		err = redisConn.Send("ZREM", attribute, id)
		if err != nil {
			redisLogger.WithFields(logrus.Fields{
				"cmd":       "ZREM",
				"attribute": attribute,
				"id":        id,
				"error":     err.Error(),
			}).Error("failed to deindex attribute")
			return status.Errorf(codes.Internal, "%v", err)
		}
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

// FilterTickets returns the Ticket ids and required attribute key-value pairs for the Tickets meeting the specified filtering criteria.
// map[ticket.Id]map[attributeName][attributeValue]
// {
//  "testplayer1": {"ranking" : 56, "loyalty_level": 4},
//  "testplayer2": {"ranking" : 50, "loyalty_level": 3},
// }
func (rb *redisBackend) FilterTickets(ctx context.Context, filters []*pb.Filter) (map[string]map[string]int64, error) {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		redisLogger.WithError(err).Error("Failed to get redis connection with context.")
	}

	defer func() {
		if err := redisConn.Close(); err != nil {
			redisLogger.WithError(err).Error("Failed to close Redis connection.")
		}
	}()

	// A map[attribute]map[playerID]value
	attributeToTickets := make(map[string]map[string]int64)
	// A set of playerIds that satisfies all filters
	idSet := make([]string, 0)

	// For each filter, do a range query to Redis on Filter.Attribute
	for i, filter := range filters {
		// Time Complexity O(logN + M), where N is the number of elements in the attribute set
		// and M is the number of entries being returned.
		// TODO: discuss if we need a LIMIT for # of queries being returned
		idToValue, err := redis.Int64Map(redisConn.Do("ZRANGEBYSCORE", filter.Attribute, filter.Min, filter.Max, "WITHSCORES"))
		if err != nil {
			redisLogger.WithFields(logrus.Fields{
				"Command": fmt.Sprintf("ZRANGEBYSCORE %s %f %f WITHSCORES", filter.Attribute, filter.Min, filter.Max),
			}).WithError(err)
			return nil, err
		}

		attributeToTickets[filter.Attribute] = idToValue

		idsPerFilter := make([]string, 0)
		for id := range idToValue {
			idsPerFilter = append(idsPerFilter, id)
		}

		if i == 0 {
			idSet = idsPerFilter
		} else {
			idSet = set.Intersection(idSet, idsPerFilter)
		}
	}

	// Result is a mapping from ticket ids to attribute key-value pairs
	results := make(map[string]map[string]int64)
	for _, id := range idSet {
		// A map from attribute names to attribute values per ticket
		propertyPerTicket := make(map[string]int64)

		for attr, tickets := range attributeToTickets {
			attrVal := tickets[id]
			propertyPerTicket[attr] = attrVal
		}
		results[id] = propertyPerTicket
	}

	return results, nil
}

func handleConnectionClose(conn *redis.Conn) {
	err := (*conn).Close()
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"error": err,
		}).Debug("failed to close redis client connection.")
	}
}

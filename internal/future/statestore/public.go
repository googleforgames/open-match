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

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/future/pb"
)

var (
	publicLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "statestore.public",
	})
)

// Service is a generic interface for talking to a storage backend.
type Service interface {
	// CreateTicket creates a new Ticket in the state storage. This method fails if the Ticket already exists.
	CreateTicket(ctx context.Context, ticket pb.Ticket, ttl int) error

	// GetTicket gets the Ticket with the specified id from state storage. This method fails if the Ticket does not exist.
	GetTicket(ctx context.Context, id string) (*pb.Ticket, error)

	// DeleteTicket removes the Ticket with the specified id from state storage. This method succeeds if the Ticket does not exist.
	DeleteTicket(ctx context.Context, id string) error

	// IndexTicket indexes the Ticket id for the specified fields.
	IndexTicket(ctx context.Context, ticket pb.Ticket, indices []string) error

	// DeindexTicket removes the indexing for the specified Ticket. Only the indexes are removed but the Ticket continues to exist.
	DeindexTicket(ctx context.Context, id string, indices []string) error

	// FilterTickets returns the Ticket ids for the Tickets meeting the specified filtering criteria.
	FilterTickets(ctx context.Context, filters []pb.Filter) ([]string, error)

	// Closes the connection to the underlying storage.
	Close() error
}

// New creates a Service based on the configuration.
func New(cfg config.View) (Service, error) {
	return newRedis(cfg)
}

// GetDeprecatedRedisPool returns a Redigo Redis pool.
// This method is used to allow code to be ported over to the new storage.Service APIs iteratively.
// This method will be going away after the 0.6.0 release.
func GetDeprecatedRedisPool(service Service) *redis.Pool {
	rb, ok := service.(*redisBackend)
	if !ok {
		publicLogger.Warningf("storage.Service was assumed to be *redisBackend but was %T instead, returning nil.", service)
		return nil
	}
	return rb.redisPool
}

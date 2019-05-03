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

package storage

import (
	"github.com/GoogleCloudPlatform/open-match/internal/config"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
)

var (
	publicLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "storage.public",
	})
)

// Service is a generic interface for talking to a storage backend.
type Service interface {
	// Close shutsdown the database connection.
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

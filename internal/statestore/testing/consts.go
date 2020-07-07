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

package testing

import "time"

const (
	// PoolMaxIdle is the number of maximum idle redis connections in a redis pool
	PoolMaxIdle = 5
	// PoolMaxActive is the number of maximum active redis connections in a redis pool
	PoolMaxActive = 5
	// PoolIdleTimeout is the idle duration allowance of a redis connection a a redis pool
	PoolIdleTimeout = 10 * time.Second
	// PoolHealthCheckTimeout is the read/write timeout of a healthcheck HTTP request
	PoolHealthCheckTimeout = 100 * time.Millisecond
	// pendingReleaseTimeout is the time to live duration of Open Match pending release settings
	pendingReleaseTimeout = 500 * time.Millisecond
	// InitialInterval is the initial backoff time of a backoff strategy
	InitialInterval = 30 * time.Millisecond
	// RandFactor is the randomization factor of a backoff strategy
	RandFactor = .5
	// Multiplier is the multuplier used in a exponential backoff strategy
	Multiplier = .5
	// MaxInterval is the maximum retry interval of a backoff strategy
	MaxInterval = 300 * time.Millisecond
	// MaxElapsedTime is the maximum total retry time of a backoff stragegy
	MaxElapsedTime        = 1000 * time.Millisecond
	assignedDeleteTimeout = 200 * time.Millisecond
)

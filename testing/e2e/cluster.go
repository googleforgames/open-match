//go:build e2ecluster
// +build e2ecluster

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

package e2e

import (
	"context"
	"sync"
	"testing"
	"time"

	"open-match.dev/open-match/internal/app/evaluator"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/statestore"
	mmfService "open-match.dev/open-match/testing/mmf"
)

func start(t *testing.T, eval evaluator.Evaluator, mmf mmfService.MatchFunction) (config.View, func(time.Duration)) {
	clusterLock.Lock()
	t.Cleanup(func() {
		clusterLock.Unlock()
	})
	if !clusterStarted {
		t.Fatal("Cluster not started")
	}
	clusterEval = eval
	clusterMMF = mmf

	cfg, err := config.Read()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		pool := statestore.GetRedisPool(cfg)
		conn, err := pool.GetContext(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		_, err = conn.Do("FLUSHALL")
		if err != nil {
			t.Fatal(err)
		}
		err = pool.Close()
		if err != nil {
			t.Fatal(err)
		}
	})

	return cfg, time.Sleep
}

var clusterLock sync.Mutex
var clusterEval evaluator.Evaluator
var clusterMMF mmfService.MatchFunction
var clusterStarted bool

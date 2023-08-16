//go:build !e2ecluster
// +build !e2ecluster

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
	"net"
	"strings"
	"testing"
	"time"

	"github.com/Bose/minisentinel"
	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/spf13/viper"
	"open-match.dev/open-match/internal/app/evaluator"
	"open-match.dev/open-match/internal/app/minimatch"
	"open-match.dev/open-match/internal/appmain/apptest"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/internal/telemetry"
	mmfService "open-match.dev/open-match/testing/mmf"
)

func start(t *testing.T, eval evaluator.Evaluator, mmf mmfService.MatchFunction) (config.View, func(time.Duration)) {
	mredis := miniredis.NewMiniRedis()
	err := mredis.StartAddr("localhost:0")
	if err != nil {
		t.Fatalf("failed to start miniredis, %v", err)
	}
	t.Cleanup(mredis.Close)

	msentinel := minisentinel.NewSentinel(mredis)
	err = msentinel.StartAddr("localhost:0")
	if err != nil {
		t.Fatalf("failed to start minisentinel, %v", err)
	}
	t.Cleanup(msentinel.Close)

	grpcListener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	_, grpcPort, err := net.SplitHostPort(grpcListener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	httpListener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	_, httpPort, err := net.SplitHostPort(httpListener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	listeners := []net.Listener{grpcListener, httpListener}

	cfg := viper.New()
	cfg.SetConfigType("yaml")
	err = cfg.ReadConfig(strings.NewReader(configFile))
	if err != nil {
		t.Fatal(err)
	}

	cfg.Set("redis.sentinelHostname", msentinel.Host())
	cfg.Set("redis.sentinelPort", msentinel.Port())
	cfg.Set("redis.sentinelMaster", msentinel.MasterInfo().Name)
	services := []string{apptest.ServiceName, "synchronizer", "backend", "frontend", "query", "evaluator"}
	for _, name := range services {
		cfg.Set("api."+name+".hostname", "localhost")
		cfg.Set("api."+name+".grpcport", grpcPort)
		cfg.Set("api."+name+".httpport", httpPort)
	}
	cfg.Set(rpc.ConfigNameEnableRPCLogging, *testOnlyEnableRPCLoggingFlag)
	cfg.Set("logging.level", *testOnlyLoggingLevel)
	cfg.Set(telemetry.ConfigNameEnableMetrics, *testOnlyEnableMetrics)

	apptest.TestApp(t, cfg, listeners, minimatch.BindService, mmfService.BindServiceFor(mmf), evaluator.BindServiceFor(eval))
	return cfg, mredis.FastForward
}

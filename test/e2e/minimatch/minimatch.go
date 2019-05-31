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

package minimatch

import (
	"fmt"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/app/minimatch"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"
)

const (
	minimatchPrefix   = "minimatch"
	minimatchHost     = "localhost"
	minimatchGRPCPort = "50510"
	minimatchHTTPPort = "51510"
)

// Server is a test server that serves all core Open Match components.
type Server struct {
	cfg       config.View
	rpcserver *rpc.Server
}

// GetFrontendClient returns a grpc client for Open Match frontned.
func (s *Server) GetFrontendClient() (pb.FrontendClient, error) {
	port := s.cfg.GetInt("minimatch.grpcport")
	conn, err := grpc.Dial(fmt.Sprintf(":%d", port), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return pb.NewFrontendClient(conn), nil
}

// GetBackendClient returns a grpc client for Open Match backend.
func (s *Server) GetBackendClient() (pb.BackendClient, error) {
	port := s.cfg.GetInt("minimatch.grpcport")
	conn, err := grpc.Dial(fmt.Sprintf(":%d", port), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return pb.NewBackendClient(conn), nil
}

// GetMMLogicClient returns a grpc client for Open Match mmlogic api.
func (s *Server) GetMMLogicClient() (pb.MmLogicClient, error) {
	port := s.cfg.GetInt("minimatch.grpcport")
	conn, err := grpc.Dial(fmt.Sprintf(":%d", port), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return pb.NewMmLogicClient(conn), nil
}

// Stop stops the rpc server.
func (s *Server) Stop() {
	if s.rpcserver != nil {
		s.rpcserver.Stop()
	}
}

// NewMiniMatch creates and starts an OpenMatchServer context for testing.
func NewMiniMatch(cfg config.View) (*Server, error) {
	// Create the minimatch server to be initialized.
	mmServer := &Server{
		cfg: cfg,
	}

	p, err := rpc.NewServerParamsFromConfig(mmServer.cfg, minimatchPrefix)
	if err != nil {
		return nil, err
	}

	if err := minimatch.BindService(p, mmServer.cfg); err != nil {
		return nil, err
	}

	s := &rpc.Server{}
	mmServer.rpcserver = s
	waitForStart, err := s.Start(p)
	if err != nil {
		return nil, err
	}

	go func() {
		waitForStart()
	}()

	return mmServer, nil
}

func createServerConfig() (config.View, error) {
	mredis, err := miniredis.Run()
	if err != nil {
		return nil, err
	}

	// Set up the configuration for the state store that the core Open Match
	// components will use.
	cfg := viper.New()
	cfg.Set("redis.hostname", mredis.Host())
	cfg.Set("redis.port", mredis.Port())
	cfg.Set("redis.pool.maxIdle", 1000)
	cfg.Set("redis.pool.idleTimeout", time.Second)
	cfg.Set("redis.pool.maxActive", 1000)
	cfg.Set("redis.expiration", 42000)
	cfg.Set("storage.page.size", 10)

	// Set up the attributes that a ticket will be indexed for.
	cfg.Set("playerIndices", []string{
		skillattribute,
		map1attribute,
		map2attribute,
	})

	// Set up the configuration for hosting the minimatch service.
	cfg.Set("minimatch.hostname", minimatchHost)
	cfg.Set("minimatch.grpcport", minimatchGRPCPort)
	cfg.Set("minimatch.httpport", minimatchHTTPPort)

	return cfg, nil
}

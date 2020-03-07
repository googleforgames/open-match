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

import (
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_tracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/resolver"
)

// nolint: gochecknoinits
func init() {
	// Using gRPC's DNS resolver to create clients.
	// This is a workaround for load balancing gRPC applications under k8s environments.
	// See https://kubernetes.io/blog/2018/11/07/grpc-load-balancing-on-kubernetes-without-tears/ for more details.
	// https://godoc.org/google.golang.org/grpc/resolver#SetDefaultScheme
	resolver.SetDefaultScheme("dns")
}

// NewGRPCDialOptions returns the grpc DialOptions for testing internal grpc clients with loadbalancing, tracing, and logging setups
func NewGRPCDialOptions(grpcLogger *logrus.Entry) []grpc.DialOption {
	grpcLogger.Level = logrus.DebugLevel

	si := []grpc.StreamClientInterceptor{
		grpc_logrus.StreamClientInterceptor(grpcLogger),
		grpc_tracing.StreamClientInterceptor(),
	}
	ui := []grpc.UnaryClientInterceptor{
		grpc_logrus.UnaryClientInterceptor(grpcLogger),
		grpc_tracing.UnaryClientInterceptor(),
	}
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(si...)),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(ui...)),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                20 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	return opts
}

// NewGRPCServerOptions returns the grpc testing DialOptions for internal grpc servers with loadbalancing, tracing, and logging setups
func NewGRPCServerOptions(grpcLogger *logrus.Entry) []grpc.ServerOption {
	si := []grpc.StreamServerInterceptor{
		grpc_recovery.StreamServerInterceptor(),
		grpc_validator.StreamServerInterceptor(),
		grpc_tracing.StreamServerInterceptor(),
		grpc_logrus.StreamServerInterceptor(grpcLogger),
	}
	ui := []grpc.UnaryServerInterceptor{
		grpc_recovery.UnaryServerInterceptor(),
		grpc_validator.UnaryServerInterceptor(),
		grpc_tracing.UnaryServerInterceptor(),
		grpc_logrus.UnaryServerInterceptor(grpcLogger),
	}

	return []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(si...)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(ui...)),
		grpc.KeepaliveEnforcementPolicy(
			keepalive.EnforcementPolicy{
				MinTime:             10 * time.Second,
				PermitWithoutStream: true,
			},
		),
	}
}

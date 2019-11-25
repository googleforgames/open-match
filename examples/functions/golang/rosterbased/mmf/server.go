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

package mmf

import (
	"fmt"
	"net"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_tracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/resolver"
	"open-match.dev/open-match/pkg/pb"
)

var (
	mmfServerLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "server",
	})
)

func init() {
	// Using gRPC's DNS resolver to create clients.
	// This is a workaround for load balancing gRPC applications under k8s environments.
	// See https://kubernetes.io/blog/2018/11/07/grpc-load-balancing-on-kubernetes-without-tears/ for more details.
	// https://godoc.org/google.golang.org/grpc/resolver#SetDefaultScheme
	resolver.SetDefaultScheme("dns")
}

// MatchFunctionService implements pb.MatchFunctionServer, the server generated
// by compiling the protobuf, by fulfilling the pb.MatchFunctionServer interface.
type MatchFunctionService struct {
	grpc          *grpc.Server
	mmlogicClient pb.MmLogicClient
	port          int
}

func newGRPCDialOptions() []grpc.DialOption {
	grpcLogger := logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "grpc.client",
	})
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
		grpc.WithBalancerName(roundrobin.Name),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                20 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	return opts
}

func newGRPCServerOptions() []grpc.ServerOption {
	si := []grpc.StreamServerInterceptor{
		grpc_recovery.StreamServerInterceptor(),
		grpc_validator.StreamServerInterceptor(),
		grpc_tracing.StreamServerInterceptor(),
		grpc_logrus.StreamServerInterceptor(mmfServerLogger),
	}
	ui := []grpc.UnaryServerInterceptor{
		grpc_recovery.UnaryServerInterceptor(),
		grpc_validator.UnaryServerInterceptor(),
		grpc_tracing.UnaryServerInterceptor(),
		grpc_logrus.UnaryServerInterceptor(mmfServerLogger),
	}

	return []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(si...)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(ui...)),
	}
}

// Start creates and starts the Match Function server and also connects to Open
// Match's mmlogic service. This connection is used at runtime to fetch tickets
// for pools specified in MatchProfile.
func Start(mmlogicAddr string, serverPort int) error {
	conn, err := grpc.Dial(mmlogicAddr, newGRPCDialOptions()...)
	if err != nil {
		logger.Fatalf("Failed to connect to Open Match, got %v", err)
	}
	defer conn.Close()

	mmfService := MatchFunctionService{
		mmlogicClient: pb.NewMmLogicClient(conn),
	}

	server := grpc.NewServer(newGRPCServerOptions()...)
	pb.RegisterMatchFunctionServer(server, &mmfService)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
			"port":  serverPort,
		}).Error("net.Listen() error")
		return err
	}

	logger.WithFields(logrus.Fields{
		"port": serverPort,
	}).Info("TCP net listener initialized")

	logger.Info("Serving gRPC endpoint")
	err = server.Serve(ln)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("gRPC serve() error")
		return err
	}

	return nil
}

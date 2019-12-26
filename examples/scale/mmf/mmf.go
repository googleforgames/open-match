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

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"open-match.dev/open-match/pkg/pb"

	utilTesting "open-match.dev/open-match/internal/util/testing"

	"open-match.dev/open-match/examples/scale/scenarios"
)

// Run triggers execution of a MMF.
func Run() {
	activeScenario := scenarios.ActiveScenario

	conn, err := grpc.Dial(activeScenario.MmlogicAddr, utilTesting.NewGRPCDialOptions(activeScenario.Logger)...)
	if err != nil {
		activeScenario.Logger.Fatalf("Failed to connect to Open Match, got %v", err)
	}
	defer conn.Close()

	server := grpc.NewServer(utilTesting.NewGRPCServerOptions(activeScenario.Logger)...)
	pb.RegisterMatchFunctionServer(server, activeScenario)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", activeScenario.MmfServerPort))
	if err != nil {
		activeScenario.Logger.WithFields(logrus.Fields{
			"error": err.Error(),
			"port":  activeScenario.MmfServerPort,
		}).Fatal("net.Listen() error")
	}

	activeScenario.Logger.WithFields(logrus.Fields{
		"port": activeScenario.MmfServerPort,
	}).Info("TCP net listener initialized")

	activeScenario.Logger.Info("Serving gRPC endpoint")
	err = server.Serve(ln)
	if err != nil {
		activeScenario.Logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("gRPC serve() error")
	}
}

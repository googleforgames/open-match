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
)

// matchFunctionService implements pb.MatchFunctionServer, the server generated
// by compiling the protobuf, by fulfilling the pb.MatchFunctionServer interface.
type MatchFunctionService struct {
	grpc          *grpc.Server
	mmlogicClient pb.MmLogicClient
	port          int
}

func Start(mmlogicAddr string, serverPort int) error {
	conn, err := grpc.Dial(mmlogicAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatalf("Failed to connect to Open Match, got %v", err)
	}
	defer conn.Close()

	mmfService := MatchFunctionService{
		mmlogicClient: pb.NewMmLogicClient(conn),
	}

	server := grpc.NewServer()
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

	go func() {
		err := server.Serve(ln)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("gRPC serve() error")
		}
		logger.Info("serving gRPC endpoints")
	}()

	return nil
}

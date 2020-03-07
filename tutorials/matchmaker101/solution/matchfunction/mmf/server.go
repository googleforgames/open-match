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
	"log"
	"net"

	"google.golang.org/grpc"
	"open-match.dev/open-match/pkg/pb"
)

// matchFunctionService implements pb.MatchFunctionServer, the server generated
// by compiling the protobuf, by fulfilling the pb.MatchFunctionServer interface.
type MatchFunctionService struct {
	grpc               *grpc.Server
	queryServiceClient pb.QueryServiceClient
	port               int
}

// Start creates and starts the Match Function server and also connects to Open
// Match's queryService service. This connection is used at runtime to fetch tickets
// for pools specified in MatchProfile.
func Start(queryServiceAddr string, serverPort int) {
	// Connect to QueryService.
	conn, err := grpc.Dial(queryServiceAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to Open Match, got %s", err.Error())
	}
	defer conn.Close()

	mmfService := MatchFunctionService{
		queryServiceClient: pb.NewQueryServiceClient(conn),
	}

	// Create and host a new gRPC service on the configured port.
	server := grpc.NewServer()
	pb.RegisterMatchFunctionServer(server, &mmfService)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
	if err != nil {
		log.Fatalf("TCP net listener initialization failed for port %v, got %s", serverPort, err.Error())
	}

	log.Printf("TCP net listener initialized for port %v", serverPort)
	err = server.Serve(ln)
	if err != nil {
		log.Fatalf("gRPC serve failed, got %s", err.Error())
	}
}

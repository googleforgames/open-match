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

	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/serving"
	servingTesting "open-match.dev/open-match/internal/serving/testing"
	"open-match.dev/open-match/internal/util/netlistener"
	netlistenerTesting "open-match.dev/open-match/internal/util/netlistener/testing"

	"log"
	"testing"

	"google.golang.org/grpc"
)

func createLogger() *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"test": "test",
	})
}

func createClient(lh *netlistener.ListenerHolder) (pb.FrontendClient, error) {
	conn, err := grpc.Dial(lh.AddrString(), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return pb.NewFrontendClient(conn), nil
}

// TestOpenClose verifies the gRPC server can be created in process, communicated with, and shut down.
func TestOpenClose(t *testing.T) {
	serviceLh := netlistenerTesting.MustListen()
	proxyLh := netlistenerTesting.MustListen()
	fakeService := &servingTesting.FakeFrontend{}
	server := serving.NewGrpcServer(serviceLh, proxyLh, createLogger())
	defer func() {
		err := server.Stop()
		if err != nil {
			log.Printf("server.Stop() reported an error but this is ok, %s", err)
		}
	}()
	server.AddService(func(server *grpc.Server) {
		pb.RegisterFrontendServer(server, fakeService)
	})
	server.AddProxy(pb.RegisterFrontendHandler)

	err := server.Start()
	if err != nil {
		t.Fatal(err)
	}

	client, err := createClient(serviceLh)
	if err != nil {
		t.Fatal(err)
	}

	result, err := client.CreatePlayer(context.Background(), &pb.CreatePlayerRequest{})
	if err != nil {
		t.Error(err)
	}
	if result == nil {
		t.Errorf("Result %v was not successful.", result)
	}
}

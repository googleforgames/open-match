package e2e

import (
	"context"

	pb "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/GoogleCloudPlatform/open-match/internal/serving"
	servingTesting "github.com/GoogleCloudPlatform/open-match/internal/serving/testing"
	"github.com/GoogleCloudPlatform/open-match/internal/util/netlistener"
	netlistenerTesting "github.com/GoogleCloudPlatform/open-match/internal/util/netlistener/testing"
	"github.com/sirupsen/logrus"

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

package e2e

import (
	"context"
	"fmt"

	pb "github.com/googleforgames/open-match/internal/pb"
	"github.com/googleforgames/open-match/internal/serving"
	servingTesting "github.com/googleforgames/open-match/internal/serving/testing"
	"github.com/googleforgames/open-match/internal/util/netlistener"
	netlistenerTesting "github.com/googleforgames/open-match/internal/util/netlistener/testing"
	log "github.com/sirupsen/logrus"

	"net"
	"testing"

	"google.golang.org/grpc"
)

func createLogger() *log.Entry {
	return log.WithFields(log.Fields{
		"test": "test",
	})
}

func createClient(lh *netlistener.ListenerHolder) (pb.FrontendClient, error) {
	conn, err := grpc.Dial(fmt.Sprintf(":%d", lh.Number()), grpc.WithInsecure())
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

	server.Stop()

	// Re-open the port to ensure it's free.
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", serviceLh.Number()))
	if err != nil {
		t.Errorf("grpc server is still running! Result= %v Error= %s", ln, err)
	}
	result, err = client.CreatePlayer(context.Background(), &pb.CreatePlayerRequest{})
	if err == nil {
		t.Errorf("grpc server is still running! Result= %v Error= %s", result, err)
	}
	// Re-open the proxyPort to ensure it's free.
	ln, err = net.Listen("tcp6", fmt.Sprintf(":%d", proxyLh.Number()))
	if err != nil {
		t.Errorf("grpc proxy is still running! Result= %v Error= %s", ln, err)
	}

}

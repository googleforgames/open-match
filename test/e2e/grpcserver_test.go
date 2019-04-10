package e2e

import (
	"context"
	"fmt"

	pb "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/GoogleCloudPlatform/open-match/internal/serving"
	servingTesting "github.com/GoogleCloudPlatform/open-match/internal/serving/testing"
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

func createClient(port int) (pb.FrontendClient, error) {
	conn, err := grpc.Dial(fmt.Sprintf(":%d", port), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return pb.NewFrontendClient(conn), nil
}

// TestOpenClose verifies the gRPC server can be created in process, communicated with, and shut down.
func TestOpenClose(t *testing.T) {
	port := servingTesting.MustPort()
	fakeService := &servingTesting.FakeFrontend{}
	server := serving.NewGrpcServer(port, createLogger())
	server.AddService(func(server *grpc.Server) {
		pb.RegisterFrontendServer(server, fakeService)
	})

	err := server.Start()
	if err != nil {
		t.Fatal(err)
	}

	client, err := createClient(port)
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
	ln, err := net.Listen("tcp6", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Errorf("grpc server is still running! Result= %v Error= %s", ln, err)
	}
	result, err = client.CreatePlayer(context.Background(), &pb.CreatePlayerRequest{})
	if err == nil {
		t.Errorf("grpc server is still running! Result= %v Error= %s", result, err)
	}
}

package testing

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/open-match/internal/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type FakeFrontend struct {
}

// CreatePlayer is this service's implementation of the CreatePlayer gRPC method defined in frontend.proto
func (s *FakeFrontend) CreatePlayer(ctx context.Context, group *pb.CreatePlayerRequest) (*pb.CreatePlayerResponse, error) {
	return &pb.CreatePlayerResponse{}, nil
}

// DeletePlayer is this service's implementation of the DeletePlayer gRPC method defined in frontend.proto
func (s *FakeFrontend) DeletePlayer(ctx context.Context, group *pb.DeletePlayerRequest) (*pb.DeletePlayerResponse, error) {
	return nil, fmt.Errorf("not supported")
}

func (s *FakeFrontend) GetUpdates(p *pb.GetUpdatesRequest, assignmentStream pb.Frontend_GetUpdatesServer) error {
	return fmt.Errorf("not supported")
}

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

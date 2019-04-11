package testing

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/open-match/internal/pb"
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

// GetUpdates is this service's implementation of the GetUpdates gRPC method defined in frontend.proto
func (s *FakeFrontend) GetUpdates(p *pb.GetUpdatesRequest, assignmentStream pb.Frontend_GetUpdatesServer) error {
	return fmt.Errorf("not supported")
}

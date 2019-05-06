/*
Copyright 2019 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testing

import (
	"context"
	"fmt"

	"open-match.dev/open-match/internal/pb"
)

// FakeFrontend with empty method impl used for testing
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

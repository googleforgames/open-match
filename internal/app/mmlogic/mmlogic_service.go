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

package mmlogic

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/statestore"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.mmlogic.mmlogic_service",
	})
)

const (
	// Minimum number of tickets to be returned in a streamed response for QueryTickets. This value
	// will be used if page size if not configured or is configured lower than the minimum value.
	minPageSize int = 10
	// Maximum number of tickets to be returned in a streamed response for QueryTickets. This value
	// will be used if page size is configured higher than the minimum value.
	maxPageSize int = 1000
)

// The MMLogic API provides utility functions for common MMF functionality such
// as retreiving Tickets from state storage.
type mmlogicService struct {
	cfg   config.View
	store statestore.Service
}

// newMmlogic creates and initializes the mmlogic service.
func newMmlogic(cfg config.View) (*mmlogicService, error) {
	ms := &mmlogicService{
		cfg: cfg,
	}

	// Initialize the state storage interface.
	var err error
	ms.store, err = statestore.New(cfg)
	if err != nil {
		return nil, err
	}

	return ms, nil
}

// GetPoolTickets gets the list of Tickets that match every Filter in the
// specified Pool.
func (s *mmlogicService) QueryTickets(req *pb.QueryTicketsRequest, responseServer pb.MmLogic_QueryTicketsServer) error {
	ctx := responseServer.Context()
	if req.Pool == nil {
		return status.Error(codes.InvalidArgument, "pool is empty")
	}
	poolFilters := req.Pool.Filter

	callback := func(tickets []*pb.Ticket) error {
		err := responseServer.Send(&pb.QueryTicketsResponse{Ticket: tickets})
		if err != nil {
			logger.WithError(err).Error("Failed to send Redis response to grpc server")
			return status.Errorf(codes.Aborted, err.Error())
		}
		return nil
	}

	pSize := s.cfg.GetInt("storage.page.size")
	if pSize < minPageSize {
		return status.Errorf(codes.FailedPrecondition, "page size %v is lower than minimum limit of %v", pSize, minPageSize)
	}

	if pSize > maxPageSize {
		return status.Errorf(codes.FailedPrecondition, "page size %v is higher than maximum limit of %v", pSize, maxPageSize)
	}

	// Send requests to the storage service
	err := s.store.FilterTickets(ctx, poolFilters, pSize, callback)
	if err != nil {
		logger.WithError(err).Error("Failed to retrieve result from storage service.")
		return err
	}

	return nil
}

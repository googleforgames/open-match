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
	"encoding/json"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/future/pb"
	"open-match.dev/open-match/internal/future/statestore"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.mmlogic.mmlogic_service",
	})
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
	poolFilters := req.Pool.Filter

	// Send requests to the storage service
	idsToProperties, err := s.store.FilterTickets(ctx, poolFilters)
	if err != nil {
		logger.WithError(err).Error("Failed to retrieve result from storage service.")
		return status.Error(codes.Internal, err.Error())
	}

	page := []*pb.Ticket{}
	// The ith entry when iterating through the idsToProperties map
	mapIdx := 0
	// The number of tickets in a paging response
	pSize := s.cfg.GetInt("storage.page.size")

	for id, property := range idsToProperties {
		select {
		case <-ctx.Done():
			return nil
		default:
			propertyByte, err := json.Marshal(property)
			if err != nil {
				logger.WithError(err).Error("Failed to convert property map to JSON")
				return status.Errorf(codes.Internal, err.Error())
			}
			page = append(page, &pb.Ticket{Id: id, Properties: string(propertyByte)})

			mapIdx++

			endPage := mapIdx%pSize == 0 || mapIdx == len(idsToProperties)-1
			if endPage {
				// Reaches page limit; Send a stream response then reset the page
				err := responseServer.Send(&pb.QueryTicketsResponse{Ticket: page})
				if err != nil {
					logger.WithError(err).Error("Failed to send Redis response to grpc server")
					return status.Errorf(codes.Aborted, err.Error())
				}
				page = []*pb.Ticket{}
			}
		}
	}

	return nil
}

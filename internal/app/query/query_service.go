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

package query

import (
	"context"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/pkg/pb"

	"open-match.dev/open-match/internal/statestore"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.query",
	})
)

// queryService API provides utility functions for common MMF functionality such
// as retreiving Tickets from state storage.
type queryService struct {
	cfg   config.View
	store statestore.Service
}

// QueryTickets gets a list of Tickets that match all Filters of the input Pool.
//   - If the Pool contains no Filters, QueryTickets will return all Tickets in the state storage.
// QueryTickets pages the Tickets by `storage.pool.size` and stream back response.
//   - storage.pool.size is default to 1000 if not set, and has a mininum of 10 and maximum of 10000
func (s *queryService) QueryTickets(req *pb.QueryTicketsRequest, responseServer pb.QueryService_QueryTicketsServer) error {
	pool := req.GetPool()
	if pool == nil {
		return status.Error(codes.InvalidArgument, ".pool is required")
	}

	ctx := responseServer.Context()
	pSize := getPageSize(s.cfg)

	callback := func(tickets []*pb.Ticket) error {
		err := responseServer.Send(&pb.QueryTicketsResponse{Tickets: tickets})
		if err != nil {
			logger.WithError(err).Error("Failed to send Redis response to grpc server")
			return status.Errorf(codes.Aborted, err.Error())
		}
		return nil
	}

	return doQueryTickets(ctx, pool, pSize, callback, s.store)
}

func doQueryTickets(ctx context.Context, pool *pb.Pool, pageSize int, sender func(tickets []*pb.Ticket) error, store statestore.Service) error {
	// Send requests to the storage service
	err := store.FilterTickets(ctx, pool, pageSize, sender)
	if err != nil {
		logger.WithError(err).Error("Failed to retrieve result from storage service.")
		return err
	}

	return nil
}

func getPageSize(cfg config.View) int {
	const (
		name = "storage.page.size"
		// Minimum number of tickets to be returned in a streamed response for QueryTickets. This value
		// will be used if page size is configured lower than the minimum value.
		minPageSize int = 10
		// Default number of tickets to be returned in a streamed response for QueryTickets.  This value
		// will be used if page size is not configured.
		defaultPageSize int = 1000
		// Maximum number of tickets to be returned in a streamed response for QueryTickets. This value
		// will be used if page size is configured higher than the maximum value.
		maxPageSize int = 10000
	)

	if !cfg.IsSet(name) {
		return defaultPageSize
	}

	pSize := cfg.GetInt("storage.page.size")
	if pSize < minPageSize {
		logger.Infof("page size %v is lower than the minimum limit of %v", pSize, maxPageSize)
		pSize = minPageSize
	}

	if pSize > maxPageSize {
		logger.Infof("page size %v is higher than the maximum limit of %v", pSize, maxPageSize)
		return maxPageSize
	}

	return pSize
}

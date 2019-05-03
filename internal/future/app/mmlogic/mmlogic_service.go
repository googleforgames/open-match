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
	"math"
	"time"

	"encoding/json"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/future/pb"
	"open-match.dev/open-match/internal/set"

	"fmt"
)

// The MMLogic API provides utility functions for common MMF functionality such
// as retreiving Tickets from state storage.
type mmlogicService struct {
	redisPool *redis.Pool
	logger    *logrus.Entry
	cfg       config.View
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
// TODO: Consider renaming to "GetPool" to be consistent with HTTP REST CRUD
// conventions. Right now there's a GET and a POST for this verb.
func (s *mmlogicService) RetrievePool(req *pb.RetrievePoolRequest, responseServer pb.MmLogic_RetrievePoolServer) error {
	poolName := req.Pool.Name
	poolFilters := req.Pool.Filter
	redisConn := s.redisPool.Get()
	defer redisConn.Close()

	s.logger.WithFields(logrus.Fields{
		"Pool.Name":    poolName,
		"Pool.Filters": poolFilters,
	}).Debug("Received RetrievePoolRequest.")

	// A map[attribute]map[playerID]value
	selectedPlayerPools := make(map[string]map[string]int64)
	// A set of playerIds that satisfies all filters
	playerIdsSet := make([]string, 0)

	// For each filter, do a range query to Redis on Filter.Attribute
	for i, filter := range poolFilters {
		filterStart := time.Now()
		// Time Complexity O(logN + M), where N is the number of elements in the attribute set
		// and M is the number of entries being returned.
		filterPlayers, err := redis.Int64Map(redisConn.Do("ZRANGEBYSCORE", filter.Attribute, filter.Min, filter.Max, "WITHSCORES"))

		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"Command": fmt.Sprintf("ZRANGEBYSCORE %s %d %d WITHSCORES", filter.Attribute, filter.Min, filter.Max),
			}).WithError(err)
		}

		s.logger.WithFields(logrus.Fields{
			"Filter.Attribute": filter.Attribute,
			"Filter.Min":       filter.Min,
			"Filter.Max":       filter.Max,
			"TimeElapsed":      time.Since(filterStart).Seconds(),
			"ResultCounts":     len(filterPlayers),
		}).Debug("Filter Stats.")

		selectedPlayerPools[filter.Attribute] = filterPlayers

		filterPlayerKeys := make([]string, 0)
		for key := range filterPlayers {
			filterPlayerKeys = append(filterPlayerKeys, key)
		}

		if i == 0 {
			playerIdsSet = filterPlayerKeys
		} else {
			playerIdsSet = set.Intersection(playerIdsSet, filterPlayerKeys)
		}
	}

	ctx := responseServer.Context()
	pageSize := s.cfg.GetInt("redis.results.pageSize")
	pageCount := int(math.Ceil(float64(len(playerIdsSet) / pageSize)))
	setOffset := 0

	for pageIndex := 0; pageIndex < pageCount; pageIndex++ {
		// Initialize a response to send for each page
		responseTickets := []*pb.Ticket{}

		for pageOffset := 0; pageOffset < pageSize && setOffset < len(playerIdsSet); pageOffset++ {
			ticketID := playerIdsSet[setOffset]
			ticketAttribute := make(map[string]int64, 0)

			for attribute, attributePool := range selectedPlayerPools {
				ticketAttribute[attribute] = attributePool[ticketID]
			}

			ticketAttributeJSONByte, err := json.Marshal(ticketAttribute)
			ticketAttributeJSON := string(ticketAttributeJSONByte)

			if err != nil {
				s.logger.WithError(err).Error("Failed to convert go struct to JSON")
				return err
			}

			responseTickets = append(responseTickets, &pb.Ticket{Id: ticketID, Properties: ticketAttributeJSON})

			setOffset++
		}
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := responseServer.Send(&pb.RetrievePoolResponse{
				Ticket: responseTickets,
			}); err != nil {
				s.logger.WithError(err).Error("Failed to send Redis response to grpc server")
				return err
			}
		}
	}

	return nil
}

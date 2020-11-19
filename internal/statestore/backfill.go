// Copyright 2020 Google LLC
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

package statestore

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/ipb"
	"open-match.dev/open-match/pkg/pb"
)

// CreateBackfill creates a new Backfill in the state storage if one doesn't exist. The xids algorithm used to create the ids ensures that they are unique with no system wide synchronization. Calling clients are forbidden from choosing an id during create. So no conflicts will occur.
func (rb *redisBackend) CreateBackfill(ctx context.Context, backfill *pb.Backfill, ticketIDs []string) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "CreateBackfill, id: %s, failed to connect to redis: %v", backfill.GetId(), err)
	}
	defer handleConnectionClose(&redisConn)

	bf := ipb.BackfillInternal{
		Backfill:  backfill,
		TicketIds: ticketIDs,
	}

	value, err := proto.Marshal(&bf)
	if err != nil {
		err = errors.Wrapf(err, "failed to marshal the backfill proto, id: %s", backfill.GetId())
		return status.Errorf(codes.Internal, "%v", err)
	}

	res, err := redisConn.Do("SETNX", backfill.GetId(), value)
	if err != nil {
		err = errors.Wrapf(err, "failed to set the value for backfill, id: %s", backfill.GetId())
		return status.Errorf(codes.Internal, "%v", err)
	}

	if res.(int64) == 0 {
		return status.Errorf(codes.AlreadyExists, "backfill already exists, id: %s", backfill.GetId())
	}
	return nil
}

// GetBackfill gets the Backfill with the specified id from state storage. This method fails if the Backfill does not exist. Returns the Backfill and asossiated ticketIDs if they exist.
func (rb *redisBackend) GetBackfill(ctx context.Context, id string) (*pb.Backfill, []string, error) {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return nil, nil, status.Errorf(codes.Unavailable, "GetBackfill, id: %s, failed to connect to redis: %v", id, err)
	}
	defer handleConnectionClose(&redisConn)

	value, err := redis.Bytes(redisConn.Do("GET", id))
	if err != nil {
		// Return NotFound if redigo did not find the backfill in storage.
		if err == redis.ErrNil {
			return nil, nil, status.Errorf(codes.NotFound, "Backfill id: %s not found", id)
		}

		err = errors.Wrapf(err, "failed to get the backfill from state storage, id: %s", id)
		return nil, nil, status.Errorf(codes.Internal, "%v", err)
	}

	if value == nil {
		return nil, nil, status.Errorf(codes.NotFound, "Backfill id: %s not found", id)
	}

	bi := &ipb.BackfillInternal{}
	err = proto.Unmarshal(value, bi)
	if err != nil {
		err = errors.Wrapf(err, "failed to unmarshal internal backfill, id: %s", id)
		return nil, nil, status.Errorf(codes.Internal, "%v", err)
	}

	return bi.Backfill, bi.TicketIds, nil
}

// DeleteBackfill removes the Backfill with the specified id from state storage. This method succeeds if the Backfill does not exist.
func (rb *redisBackend) DeleteBackfill(ctx context.Context, id string) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "DeleteBackfill, id: %s, failed to connect to redis: %v", id, err)
	}
	defer handleConnectionClose(&redisConn)

	_, err = redisConn.Do("DEL", id)
	if err != nil {
		err = errors.Wrapf(err, "failed to delete the backfill from state storage, id: %s", id)
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

// UpdateBackfill updates an existing Backfill with a new data. ticketIDs can be nil.
func (rb *redisBackend) UpdateBackfill(ctx context.Context, backfill *pb.Backfill, ticketIDs []string) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "UpdateBackfill, id: %s, failed to connect to redis: %v", backfill.GetId(), err)
	}
	defer handleConnectionClose(&redisConn)

	bf := ipb.BackfillInternal{
		Backfill:  backfill,
		TicketIds: ticketIDs,
	}

	value, err := proto.Marshal(&bf)
	if err != nil {
		err = errors.Wrapf(err, "failed to marshal the backfill proto, id: %s", backfill.GetId())
		return status.Errorf(codes.Internal, "%v", err)
	}

	_, err = redisConn.Do("SET", backfill.GetId(), value)
	if err != nil {
		err = errors.Wrapf(err, "failed to set the value for backfill, id: %s", backfill.GetId())
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

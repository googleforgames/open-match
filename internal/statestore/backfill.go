package statestore

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/pkg/pb"
)

// CreateBackfill creates a new Backfill in the state storage. If the id already exists, it will be overwritten.
func (rb *redisBackend) CreateBackfill(ctx context.Context, backfill *pb.Backfill) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "CreateBackfill, id: %s, failed to connect to redis: %v", backfill.GetId(), err)
	}
	defer handleConnectionClose(&redisConn)

	value, err := proto.Marshal(backfill)
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

// GetBackfill gets the Backfill with the specified id from state storage. This method fails if the Backfill does not exist.
func (rb *redisBackend) GetBackfill(ctx context.Context, id string) (*pb.Backfill, error) {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "GetBackfill, id: %s, failed to connect to redis: %v", id, err)
	}
	defer handleConnectionClose(&redisConn)

	value, err := redis.Bytes(redisConn.Do("GET", id))
	if err != nil {
		// Return NotFound if redigo did not find the backfill in storage.
		if err == redis.ErrNil {
			msg := fmt.Sprintf("Backfill id: %s not found", id)
			return nil, status.Error(codes.NotFound, msg)
		}

		err = errors.Wrapf(err, "failed to get the backfill from state storage, id: %s", id)
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	if value == nil {
		msg := fmt.Sprintf("Backfill id: %s not found", id)
		return nil, status.Error(codes.NotFound, msg)
	}

	backfill := &pb.Backfill{}
	err = proto.Unmarshal(value, backfill)
	if err != nil {
		err = errors.Wrapf(err, "failed to unmarshal the backfill proto, id: %s", id)
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	return backfill, nil
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

// UpdateBackfill updates an exising Backfill with new data.
func (rb *redisBackend) UpdateBackfill(ctx context.Context, backfill *pb.Backfill, updateFunc func(current *pb.Backfill, new *pb.Backfill) (*pb.Backfill, error)) (*pb.Backfill, error) {
	if updateFunc == nil {
		return nil, status.Errorf(codes.Internal, "nil updateFunc provided")
	}

	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "UpdateBackfill, id: %s, failed to connect to redis: %v", backfill.GetId(), err)
	}
	defer handleConnectionClose(&redisConn)

	// get current backfill
	value, err := redis.Bytes(redisConn.Do("GET", backfill.GetId()))
	if err != nil {
		// Return NotFound if redigo did not find the backfill in storage.
		if err == redis.ErrNil {
			msg := fmt.Sprintf("Backfill id: %s not found", backfill.GetId())
			return nil, status.Error(codes.NotFound, msg)
		}

		err = errors.Wrapf(err, "failed to get the backfill from state storage, id: %s", backfill.GetId())
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	if value == nil {
		msg := fmt.Sprintf("Backfill id: %s not found", backfill.GetId())
		return nil, status.Error(codes.NotFound, msg)
	}

	currentBackfill := &pb.Backfill{}
	err = proto.Unmarshal(value, currentBackfill)
	if err != nil {
		err = errors.Wrapf(err, "failed to unmarshal the backfill proto, id: %s", backfill.GetId())
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	// update current backfill by invoking an updateFunc which is implemented on the caller side
	backfillToSet, err := updateFunc(currentBackfill, backfill)
	if err != nil {
		return nil, err
	}

	value, err = proto.Marshal(backfillToSet)
	if err != nil {
		err = errors.Wrapf(err, "failed to marshal the backfill proto, id: %s", backfillToSet.GetId())
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	_, err = redisConn.Do("SET", backfill.GetId(), value)
	if err != nil {
		err = errors.Wrapf(err, "failed to set the value for backfill, id: %s", backfillToSet.GetId())
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	return backfillToSet, nil
}

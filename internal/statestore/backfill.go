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
	"strconv"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/ipb"
	"open-match.dev/open-match/pkg/pb"
)

var logger = logrus.WithFields(logrus.Fields{
	"app":       "openmatch",
	"component": "statestore.redis",
})

const (
	backfillLastAckTime = "backfill_last_ack_time"
	allBackfills        = "allBackfills"
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

	return doUpdateAcknowledgmentTimestamp(redisConn, backfill.GetId())
}

// GetBackfill gets the Backfill with the specified id from state storage. This method fails if the Backfill does not exist. Returns the Backfill and associated ticketIDs if they exist.
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

// GetBackfills returns multiple backfills from storage
func (rb *redisBackend) GetBackfills(ctx context.Context, ids []string) ([]*pb.Backfill, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "GetBackfills, failed to connect to redis: %v", err)
	}
	defer handleConnectionClose(&redisConn)

	queryParams := make([]interface{}, len(ids))
	for i, id := range ids {
		queryParams[i] = id
	}

	slices, err := redis.ByteSlices(redisConn.Do("MGET", queryParams...))
	if err != nil {
		err = errors.Wrapf(err, "failed to lookup backfills: %v", ids)
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	m := make(map[string]*pb.Backfill, len(ids))
	for i, s := range slices {
		if s != nil {
			b := &ipb.BackfillInternal{}
			err = proto.Unmarshal(s, b)
			if err != nil {
				err = errors.Wrapf(err, "failed to unmarshal backfill from redis, key: %s", ids[i])
				return nil, status.Errorf(codes.Internal, "%v", err)
			}

			if b.Backfill != nil {
				m[b.Backfill.Id] = b.Backfill
			}
		}
	}

	var notFound []string
	result := make([]*pb.Backfill, 0, len(ids))
	for _, id := range ids {
		if b, ok := m[id]; ok {
			result = append(result, b)
		} else {
			notFound = append(notFound, id)
		}
	}

	if len(notFound) > 0 {
		redisLogger.Warningf("failed to lookup backfills: %v", notFound)
	}

	return result, nil
}

// DeleteBackfill removes the Backfill with the specified id from state storage.
func (rb *redisBackend) DeleteBackfill(ctx context.Context, id string) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "DeleteBackfill, id: %s, failed to connect to redis: %v", id, err)
	}
	defer handleConnectionClose(&redisConn)

	value, err := redis.Int(redisConn.Do("DEL", id))
	if err != nil {
		err = errors.Wrapf(err, "failed to delete the backfill from state storage, id: %s", id)
		return status.Errorf(codes.Internal, "%v", err)
	}

	if value == 0 {
		return status.Errorf(codes.NotFound, "Backfill id: %s not found", id)
	}

	return rb.deleteExpiredBackfillID(redisConn, id)
}

// UpdateBackfill updates an existing Backfill with a new data. ticketIDs can be nil.
func (rb *redisBackend) UpdateBackfill(ctx context.Context, backfill *pb.Backfill, ticketIDs []string) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "UpdateBackfill, id: %s, failed to connect to redis: %v", backfill.GetId(), err)
	}
	defer handleConnectionClose(&redisConn)

	expired, err := isBackfillExpired(redisConn, backfill.Id, getBackfillReleaseTimeoutFraction(rb.cfg))
	if err != nil {
		return err
	}

	if expired {
		return status.Errorf(codes.FailedPrecondition, "can not update an expired backfill, id: %s", backfill.Id)
	}

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

func isBackfillExpired(conn redis.Conn, id string, ttl time.Duration) (bool, error) {
	lastAckTime, err := redis.Float64(conn.Do("ZSCORE", backfillLastAckTime, id))
	if err != nil {
		return false, status.Errorf(codes.Internal, "%v",
			errors.Wrapf(err, "failed to get backfill's last acknowledgement time, id: %s", id))
	}

	endTime := time.Now().Add(-ttl).UnixNano()
	return int64(lastAckTime) < endTime, nil
}

// DeleteBackfillCompletely performs a set of operations to remove backfill and all related entities.
func (rb *redisBackend) DeleteBackfillCompletely(ctx context.Context, id string) error {
	m := rb.NewMutex(id)
	err := m.Lock(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if _, err = m.Unlock(context.Background()); err != nil {
			logger.WithError(err).Error("error on mutex unlock")
		}
	}()

	// 1. deindex backfill
	err = rb.DeindexBackfill(ctx, id)
	if err != nil {
		return err
	}

	// just log errors and try to perform as mush actions as possible

	// 2. get associated with a current backfill tickets ids
	_, associatedTickets, err := rb.GetBackfill(ctx, id)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":       err.Error(),
			"backfill_id": id,
		}).Error("DeleteBackfillCompletely - failed to GetBackfill")
	}

	// 3. delete associated tickets from pending release state
	err = rb.DeleteTicketsFromPendingRelease(ctx, associatedTickets)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":       err.Error(),
			"backfill_id": id,
		}).Error("DeleteBackfillCompletely - failed to DeleteTicketsFromPendingRelease")
	}

	// 4. delete backfill
	err = rb.DeleteBackfill(ctx, id)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":       err.Error(),
			"backfill_id": id,
		}).Error("DeleteBackfillCompletely - failed to DeleteBackfill")
	}

	return nil
}

func (rb *redisBackend) cleanupWorker(ctx context.Context, backfillIDsCh <-chan string, wg *sync.WaitGroup) {
	var err error
	for id := range backfillIDsCh {
		err = rb.DeleteBackfillCompletely(ctx, id)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"error":       err.Error(),
				"backfill_id": id,
			}).Error("CleanupBackfills")
		}
		wg.Done()
	}
}

// CleanupBackfills removes expired backfills
func (rb *redisBackend) CleanupBackfills(ctx context.Context) error {
	expiredBfIDs, err := rb.GetExpiredBackfillIDs(ctx)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(expiredBfIDs))
	backfillIDsCh := make(chan string, len(expiredBfIDs))

	for w := 1; w <= 3; w++ {
		go rb.cleanupWorker(ctx, backfillIDsCh, &wg)
	}

	for _, id := range expiredBfIDs {
		backfillIDsCh <- id
	}
	close(backfillIDsCh)

	wg.Wait()
	return nil
}

// UpdateAcknowledgmentTimestamp stores Backfill's last acknowledgement time.
// Check on Backfill existence should be performed on Frontend side
func (rb *redisBackend) UpdateAcknowledgmentTimestamp(ctx context.Context, id string) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "UpdateAcknowledgmentTimestamp, id: %s, failed to connect to redis: %v", id, err)
	}
	defer handleConnectionClose(&redisConn)

	expired, err := isBackfillExpired(redisConn, id, getBackfillReleaseTimeoutFraction(rb.cfg))
	if err != nil {
		return err
	}

	if expired {
		return status.Errorf(codes.FailedPrecondition, "can not acknowledge an expired backfill, id: %s", id)
	}

	return doUpdateAcknowledgmentTimestamp(redisConn, id)
}

func doUpdateAcknowledgmentTimestamp(conn redis.Conn, backfillID string) error {
	currentTime := time.Now().UnixNano()

	_, err := conn.Do("ZADD", backfillLastAckTime, currentTime, backfillID)
	if err != nil {
		return status.Errorf(codes.Internal, "%v",
			errors.Wrap(err, "failed to store backfill's last acknowledgement time"))
	}

	return nil
}

// GetExpiredBackfillIDs gets all backfill IDs which are expired
func (rb *redisBackend) GetExpiredBackfillIDs(ctx context.Context) ([]string, error) {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "GetExpiredBackfillIDs, failed to connect to redis: %v", err)
	}
	defer handleConnectionClose(&redisConn)

	ttl := getBackfillReleaseTimeoutFraction(rb.cfg)
	curTime := time.Now()
	endTimeInt := curTime.Add(-ttl).UnixNano()
	startTimeInt := 0

	// Filter out backfill IDs that are fetched but not assigned within TTL time (ms).
	expiredBackfillIds, err := redis.Strings(redisConn.Do("ZRANGEBYSCORE", backfillLastAckTime, startTimeInt, endTimeInt))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting expired backfills %v", err)
	}

	return expiredBackfillIds, nil
}

// deleteExpiredBackfillID deletes expired BackfillID from a sorted set
func (rb *redisBackend) deleteExpiredBackfillID(conn redis.Conn, backfillID string) error {
	_, err := conn.Do("ZREM", backfillLastAckTime, backfillID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to delete expired backfill ID %s from Sorted Set %s",
			backfillID, err.Error())
	}
	return nil
}

// IndexBackfill adds the backfill to the index.
func (rb *redisBackend) IndexBackfill(ctx context.Context, backfill *pb.Backfill) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "IndexBackfill, id: %s, failed to connect to redis: %v", backfill.GetId(), err)
	}
	defer handleConnectionClose(&redisConn)

	err = redisConn.Send("HSET", allBackfills, backfill.Id, backfill.Generation)
	if err != nil {
		err = errors.Wrapf(err, "failed to add backfill to all backfills, id: %s", backfill.Id)
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

// DeindexBackfill removes specified Backfill ID from the index. The Backfill continues to exist.
func (rb *redisBackend) DeindexBackfill(ctx context.Context, id string) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "DeindexBackfill, id: %s, failed to connect to redis: %v", id, err)
	}
	defer handleConnectionClose(&redisConn)

	err = redisConn.Send("HDEL", allBackfills, id)
	if err != nil {
		err = errors.Wrapf(err, "failed to remove ID from backfill index, id: %s", id)
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

// GetIndexedBackfills returns the ids of all backfills currently indexed.
func (rb *redisBackend) GetIndexedBackfills(ctx context.Context) (map[string]int, error) {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "GetIndexedBackfills, failed to connect to redis: %v", err)
	}
	defer handleConnectionClose(&redisConn)

	ttl := getBackfillReleaseTimeoutFraction(rb.cfg)
	curTime := time.Now()
	endTimeInt := curTime.Add(time.Hour).UnixNano()
	startTimeInt := curTime.Add(-ttl).UnixNano()

	// Exclude expired backfills
	acknowledgedIds, err := redis.Strings(redisConn.Do("ZRANGEBYSCORE", backfillLastAckTime, startTimeInt, endTimeInt))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting acknowledged backfills %v", err)
	}

	index, err := redis.StringMap(redisConn.Do("HGETALL", allBackfills))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting all indexed backfill ids %v", err)
	}

	r := make(map[string]int, len(acknowledgedIds))
	for _, id := range acknowledgedIds {
		if generation, ok := index[id]; ok {
			gen, err := strconv.Atoi(generation)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "error while parsing generation into number: %v", err)
			}
			r[id] = gen
		}
	}

	return r, nil
}

func getBackfillReleaseTimeout(cfg config.View) time.Duration {
	const (
		name = "pendingReleaseTimeout"
		// Default timeout to release backfill. This value
		// will be used if pendingReleaseTimeout is not configured.
		defaultpendingReleaseTimeout time.Duration = 1 * time.Minute
	)

	if !cfg.IsSet(name) {
		return defaultpendingReleaseTimeout
	}

	return cfg.GetDuration(name)
}

func getBackfillReleaseTimeoutFraction(cfg config.View) time.Duration {
	// Use a fraction 80% of pendingRelease Tickets TTL
	ttl := getBackfillReleaseTimeout(cfg) / 5 * 4
	return ttl
}

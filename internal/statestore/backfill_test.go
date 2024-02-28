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
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"open-match.dev/open-match/internal/config"
	utilTesting "open-match.dev/open-match/internal/util/testing"
	"open-match.dev/open-match/pkg/pb"
)

func TestCreateBackfillLastAckTime(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	bfID := "1234"
	ctx := utilTesting.NewContext(t)
	err := service.CreateBackfill(ctx, &pb.Backfill{
		Id: bfID,
	}, nil)
	require.NoError(t, err)

	pool := GetRedisPool(cfg)
	conn := pool.Get()

	// test that Backfill last acknowledged is in a sorted set
	ts, redisErr := redis.Int64(conn.Do("ZSCORE", backfillLastAckTime, bfID))
	require.NoError(t, redisErr)
	require.True(t, ts > 0, "timestamp is not valid")
}

func TestCreateBackfill(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	bf := pb.Backfill{
		Id:         "1",
		Generation: 1,
	}

	testCases := []struct {
		description     string
		backfill        *pb.Backfill
		ticketIDs       []string
		expectedCode    codes.Code
		expectedMessage string
	}{
		{
			description:     "ok, backfill is passed, ticketIDs is nil",
			backfill:        &bf,
			ticketIDs:       []string{"1", "2"},
			expectedCode:    codes.OK,
			expectedMessage: "",
		},
		{
			description:     "create existing backfill, err expected",
			backfill:        &bf,
			ticketIDs:       nil,
			expectedCode:    codes.AlreadyExists,
			expectedMessage: "backfill already exists, id: 1",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			err := service.CreateBackfill(ctx, tc.backfill, tc.ticketIDs)
			if tc.expectedCode == codes.OK {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Equal(t, tc.expectedCode.String(), status.Convert(err).Code().String())
				require.Contains(t, status.Convert(err).Message(), tc.expectedMessage)
			}
		})
	}

	// pass an expired context, err expected
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	err := service.CreateBackfill(ctx, &pb.Backfill{
		Id: "222",
	}, nil)
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "CreateBackfill, id: 222, failed to connect to redis:")
}

func TestUpdateExistingBackfillNoError(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	// ARRANGE
	v := &wrapperspb.DoubleValue{Value: 123}
	a, err := anypb.New(v)
	require.NoError(t, err)

	existingBF := pb.Backfill{
		Id:         "123",
		Generation: 1,
		SearchFields: &pb.SearchFields{
			Tags: []string{"123"},
		},
		Extensions: map[string]*anypb.Any{
			"qwe": a,
		},
	}
	ticketIDs := []string{"1"}
	err = service.CreateBackfill(ctx, &existingBF, ticketIDs)
	require.NoError(t, err)

	updateBF := pb.Backfill{
		Id:         existingBF.Id,
		Generation: 5,
		SearchFields: &pb.SearchFields{
			Tags: []string{"456"},
		},
		Extensions: map[string]*anypb.Any{
			"xyz": a,
		},
	}
	updateTicketIDs := []string{"1"}

	// ACT
	err = service.UpdateBackfill(ctx, &updateBF, updateTicketIDs)
	require.NoError(t, err)

	// ASSERT
	backfillActual, tIDsActual, err := service.GetBackfill(ctx, updateBF.Id)
	require.NoError(t, err)

	require.Equal(t, updateTicketIDs, tIDsActual)
	require.Equal(t, updateBF.Id, backfillActual.Id)
	require.Equal(t, updateBF.Generation, backfillActual.Generation)

	require.NotNil(t, backfillActual.SearchFields)
	require.Equal(t, updateBF.SearchFields.Tags, backfillActual.SearchFields.Tags)

	res := &wrapperspb.DoubleValue{}
	err = backfillActual.Extensions["xyz"].UnmarshalTo(res)
	require.NoError(t, err)
	require.Equal(t, v.Value, res.Value)
}

func TestUpdateBackfillDoNotExistCanNotUpdate(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	v := &wrapperspb.DoubleValue{Value: 123}
	a, err := anypb.New(v)
	require.NoError(t, err)

	updateBF := pb.Backfill{
		Id:         "123",
		Generation: 5,
		SearchFields: &pb.SearchFields{
			Tags: []string{"456"},
		},
		Extensions: map[string]*anypb.Any{
			"xyz": a,
		},
	}
	updateTicketIDs := []string{"1"}

	err = service.UpdateBackfill(ctx, &updateBF, updateTicketIDs)
	require.Error(t, err)
	require.Equal(t, codes.Internal.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "failed to get backfill's last acknowledgement time, id: 123")
}

func TestUpdateBackfillExpiredBackfillErrExpected(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	rc, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port")))
	require.NoError(t, err)

	bfID := "bf1"
	bfLastAck := "backfill_last_ack_time"
	bf := pb.Backfill{
		Id:         bfID,
		Generation: 5,
	}

	// add expired but acknowledged backfill
	_, err = rc.Do("ZADD", bfLastAck, 123, bfID)
	require.NoError(t, err)

	err = service.UpdateBackfill(ctx, &bf, nil)
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), fmt.Sprintf("can not update an expired backfill, id: %s", bfID))
}

func TestUpdateBackfillExpiredContextErrExpected(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	err := service.UpdateBackfill(ctx, &pb.Backfill{
		Id: "222",
	}, nil)
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "UpdateBackfill, id: 222, failed to connect to redis:")
}

func TestGetBackfill(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	expectedBackfill := &pb.Backfill{
		Id:         "mockBackfillID",
		Generation: 1,
	}
	expectedTicketIDs := []string{"1", "2"}
	err := service.CreateBackfill(ctx, expectedBackfill, expectedTicketIDs)
	require.NoError(t, err)

	c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port")))
	require.NoError(t, err)
	_, err = c.Do("SET", "wrong-type-key", "wrong-type-value")
	require.NoError(t, err)

	testCases := []struct {
		description     string
		backfillID      string
		expectedCode    codes.Code
		expectedMessage string
	}{
		{
			description:     "backfill is found",
			backfillID:      "mockBackfillID",
			expectedCode:    codes.OK,
			expectedMessage: "",
		},
		{
			description:     "empty id passed, err expected",
			backfillID:      "",
			expectedCode:    codes.NotFound,
			expectedMessage: "Backfill id:  not found",
		},
		{
			description:     "wrong id passed, err expected",
			backfillID:      "123456",
			expectedCode:    codes.NotFound,
			expectedMessage: "Backfill id: 123456 not found",
		},
		{
			description:     "item of a wrong type is requested, err expected",
			backfillID:      "wrong-type-key",
			expectedCode:    codes.Internal,
			expectedMessage: "failed to unmarshal internal backfill, id: wrong-type-key:",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			backfillActual, tidsActual, errActual := service.GetBackfill(ctx, tc.backfillID)
			if tc.expectedCode == codes.OK {
				require.NoError(t, errActual)
				require.NotNil(t, backfillActual)
				require.Equal(t, expectedBackfill.Id, backfillActual.Id)
				require.Equal(t, expectedBackfill.SearchFields, backfillActual.SearchFields)
				require.Equal(t, expectedBackfill.Extensions, backfillActual.Extensions)
				require.Equal(t, expectedBackfill.Generation, backfillActual.Generation)
				require.Equal(t, expectedTicketIDs, tidsActual)
			} else {
				require.Nil(t, backfillActual)
				require.Nil(t, tidsActual)
				require.Error(t, errActual)
				require.Equal(t, tc.expectedCode.String(), status.Convert(errActual).Code().String())
				require.Contains(t, status.Convert(errActual).Message(), tc.expectedMessage)
			}
		})
	}

	// pass an expired context, err expected
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	bf, tids, err := service.GetBackfill(ctx, "12345")
	require.Error(t, err)
	require.Nil(t, bf)
	require.Nil(t, tids)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "GetBackfill, id: 12345, failed to connect to redis:")
}

func TestDeleteBackfill(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	// Last Acknowledge timestamp is updated on Frontend CreateBackfill
	bfID := "mockBackfillID"
	err := service.CreateBackfill(ctx, &pb.Backfill{
		Id:         bfID,
		Generation: 1,
	}, nil)
	require.NoError(t, err)

	pool := GetRedisPool(cfg)
	conn := pool.Get()

	ts, err := redis.Int64(conn.Do("ZSCORE", backfillLastAckTime, bfID))
	require.NoError(t, err)
	require.True(t, ts > 0, "timestamp is not valid")

	testCases := []struct {
		description     string
		backfillID      string
		expectedCode    codes.Code
		expectedMessage string
	}{
		{
			description:     "backfill is found and deleted",
			backfillID:      bfID,
			expectedCode:    codes.OK,
			expectedMessage: "",
		},
		{
			description:     "empty id passed, err expected",
			backfillID:      "",
			expectedCode:    codes.NotFound,
			expectedMessage: "Backfill id:  not found",
		},
		{
			description:     "wrong id passed, err expected",
			backfillID:      "123456",
			expectedCode:    codes.NotFound,
			expectedMessage: "Backfill id: 123456 not found",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			errActual := service.DeleteBackfill(ctx, tc.backfillID)
			if tc.expectedCode == codes.OK {
				require.NoError(t, errActual)

				_, errGetTicket := service.GetTicket(ctx, tc.backfillID)
				require.Error(t, errGetTicket)
				require.Equal(t, codes.NotFound.String(), status.Convert(errGetTicket).Code().String())
				// test that Backfill also deleted from last acknowledged sorted set
				_, err = redis.Int64(conn.Do("ZSCORE", backfillLastAckTime, tc.backfillID))
				require.Error(t, err)
				require.Equal(t, err.Error(), "redigo: nil returned")
			} else {
				require.Error(t, errActual)
				require.Equal(t, tc.expectedCode.String(), status.Convert(errActual).Code().String())
				require.Contains(t, status.Convert(errActual).Message(), tc.expectedMessage)
			}
		})
	}

	// pass an expired context, err expected
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	err = service.DeleteBackfill(ctx, "12345")
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "DeleteBackfill, id: 12345, failed to connect to redis:")
}

// TestUpdateAcknowledgmentTimestampLifecycle test statestore functions - UpdateAcknowledgmentTimestamp, GetExpiredBackfillIDs
// and deleteExpiredBackfillID
func TestUpdateAcknowledgmentTimestampLifecycle(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()

	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	bf1 := "mockBackfillID"
	bf2 := "mockBackfillID2"
	err := service.CreateBackfill(ctx, &pb.Backfill{
		Id:         bf1,
		Generation: 1,
	}, nil)
	require.NoError(t, err)

	err = service.CreateBackfill(ctx, &pb.Backfill{
		Id:         bf2,
		Generation: 1,
	}, nil)
	require.NoError(t, err)

	bfIDs, err := service.GetExpiredBackfillIDs(ctx)
	require.NoError(t, err)
	require.Len(t, bfIDs, 0)
	pendingReleaseTimeout := cfg.GetDuration("pendingReleaseTimeout")

	// Sleep till all Backfills expire
	time.Sleep(pendingReleaseTimeout)

	// This call also sets initial LastAcknowledge time
	bfIDs, err = service.GetExpiredBackfillIDs(ctx)
	require.NoError(t, err)
	require.Len(t, bfIDs, 2)
	require.Contains(t, bfIDs, bf1)
	require.Contains(t, bfIDs, bf2)

	err = service.UpdateAcknowledgmentTimestamp(ctx, bf1)
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), fmt.Sprintf("can not acknowledge an expired backfill, id: %s", bf1))

	err = service.UpdateAcknowledgmentTimestamp(ctx, bf2)
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), fmt.Sprintf("can not acknowledge an expired backfill, id: %s", bf2))

	err = service.DeleteBackfill(ctx, bfIDs[0])
	require.NoError(t, err)

	bfIDs, err = service.GetExpiredBackfillIDs(ctx)
	require.Len(t, bfIDs, 1)
	require.NoError(t, err)
}

func TestUpdateAcknowledgmentTimestamp(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()

	startTime := time.Now()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)
	bf1 := "mockBackfillID"

	err := service.CreateBackfill(ctx, &pb.Backfill{
		Id:         bf1,
		Generation: 1,
	}, nil)
	require.NoError(t, err)

	err = service.UpdateAcknowledgmentTimestamp(ctx, bf1)
	require.NoError(t, err)

	// Check that Acknowledge timestamp stored valid in Redis
	pool := GetRedisPool(cfg)
	conn := pool.Get()
	res, err := redis.Int64(conn.Do("ZSCORE", backfillLastAckTime, bf1))
	require.NoError(t, err)
	// Create a time.Time from Unix nanoseconds and make sure, that time difference
	// is less than one second
	t2 := time.Unix(res/1e9, res%1e9)
	require.True(t, t2.After(startTime), "UpdateAcknowledgmentTimestamp should update time to a more recent one")
}

func TestUpdateAcknowledgmentTimestamptExpiredBackfillErrExpected(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	rc, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port")))
	require.NoError(t, err)

	bfID := "bf1"
	bfLastAck := "backfill_last_ack_time"

	// add expired but acknowledged backfill
	_, err = rc.Do("ZADD", bfLastAck, 123, bfID)
	require.NoError(t, err)

	err = service.UpdateAcknowledgmentTimestamp(ctx, bfID)
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), fmt.Sprintf("can not acknowledge an expired backfill, id: %s", bfID))
}

func TestUpdateAcknowledgmentTimestampConnectionError(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	bf1 := "mockBackfill"
	ctx := utilTesting.NewContext(t)
	cfg = createInvalidRedisConfig()
	service = New(cfg)
	require.NotNil(t, service)
	err := service.UpdateAcknowledgmentTimestamp(ctx, bf1)
	require.Error(t, err, "failed to connect to redis:")
}

func createInvalidRedisConfig() config.View {
	cfg := viper.New()

	cfg.Set("redis.hostname", "localhost")
	cfg.Set("redis.port", 222)
	return cfg
}

// TestGetExpiredBackfillIDs test statestore function GetExpiredBackfillIDs
func TestGetExpiredBackfillIDs(t *testing.T) {
	// Prepare expired and normal BackfillIds in a Redis Sorted Set
	cfg, closer := createRedis(t, false, "")
	defer closer()

	expID := "expired"
	goodID := "fresh"
	pool := GetRedisPool(cfg)
	conn := pool.Get()
	_, err := conn.Do("ZADD", backfillLastAckTime, 123, expID)
	require.NoError(t, err)
	_, err = conn.Do("ZADD", backfillLastAckTime, time.Now().UnixNano(), goodID)
	require.NoError(t, err)

	// GetExpiredBackfillIDs should return only expired BF
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)
	bfIDs, err := service.GetExpiredBackfillIDs(ctx)
	require.NoError(t, err)
	require.Len(t, bfIDs, 1)
	require.Equal(t, expID, bfIDs[0])
}

func TestIndexBackfill(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()

	t.Run("WithValidContext", func(t *testing.T) {
		ctx := utilTesting.NewContext(t)
		generateBackfills(ctx, t, service, 2)
		c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port")))
		require.NoError(t, err)
		idsIndexed, err := redis.Strings(c.Do("HKEYS", allBackfills))
		require.NoError(t, err)
		require.Len(t, idsIndexed, 2)
		require.Equal(t, "mockBackfillID-0", idsIndexed[0])
		require.Equal(t, "mockBackfillID-1", idsIndexed[1])
	})

	t.Run("WithCancelledContext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		service = New(cfg)
		err := service.IndexBackfill(ctx, &pb.Backfill{
			Id:         "12345",
			Generation: 42,
		})
		require.Error(t, err)
		require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
		require.Contains(t, status.Convert(err).Message(), "IndexBackfill, id: 12345, failed to connect to redis:")
	})
}

func TestDeindexBackfill(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()

	ctx := utilTesting.NewContext(t)

	generateBackfills(ctx, t, service, 2)

	c, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port")))
	require.NoError(t, err)
	idsIndexed, err := redis.Strings(c.Do("HKEYS", allBackfills))
	require.NoError(t, err)
	require.Len(t, idsIndexed, 2)
	require.Equal(t, "mockBackfillID-0", idsIndexed[0])
	require.Equal(t, "mockBackfillID-1", idsIndexed[1])

	// deindex and check that there is only 1 backfill in the returned slice
	err = service.DeindexBackfill(ctx, "mockBackfillID-1")
	require.NoError(t, err)
	idsIndexed, err = redis.Strings(c.Do("HKEYS", allBackfills))
	require.NoError(t, err)
	require.Len(t, idsIndexed, 1)
	require.Equal(t, "mockBackfillID-0", idsIndexed[0])

	// pass an expired context, err expected
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service = New(cfg)
	err = service.DeindexBackfill(ctx, "12345")
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "DeindexBackfill, id: 12345, failed to connect to redis:")
}

func TestGetIndexedBackfills(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()

	ctx := utilTesting.NewContext(t)

	verifyBackfills := func(service Service, backfills []*pb.Backfill) {
		ids, err := service.GetIndexedBackfills(ctx)
		require.Nil(t, err)
		require.Equal(t, len(backfills), len(ids))

		for _, bf := range backfills {
			gen, ok := ids[bf.GetId()]
			require.Equal(t, bf.Generation, int64(gen))
			require.True(t, ok)
		}
	}

	// no indexed backfills exists
	verifyBackfills(service, []*pb.Backfill{})

	// two indexed backfills exists
	backfills := generateBackfills(ctx, t, service, 2)
	verifyBackfills(service, backfills)

	// deindex one backfill, one backfill exist
	err := service.DeindexBackfill(ctx, backfills[0].Id)
	require.Nil(t, err)
	verifyBackfills(service, backfills[1:2])
}

func generateBackfills(ctx context.Context, t *testing.T, service Service, amount int) []*pb.Backfill {
	backfills := make([]*pb.Backfill, 0, amount)

	for i := 0; i < amount; i++ {
		tmp := &pb.Backfill{
			Id:         fmt.Sprintf("mockBackfillID-%d", i),
			Generation: 1,
		}
		require.NoError(t, service.CreateBackfill(ctx, tmp, []string{}))
		require.NoError(t, service.IndexBackfill(ctx, tmp))
		backfills = append(backfills, tmp)
	}

	return backfills
}

func BenchmarkCleanupBackfills(b *testing.B) {
	t := &testing.T{}
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	rc, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port")))
	require.NoError(t, err)

	createStaleBF := func(bfID string, ticketIDs ...string) {
		bf := &pb.Backfill{
			Id:         bfID,
			Generation: 1,
		}
		err = service.CreateBackfill(ctx, bf, ticketIDs)
		require.NoError(t, err)

		_, err = rc.Do("ZADD", "backfill_last_ack_time", 123, bfID)
		require.NoError(t, err)

		err = service.AddTicketsToPendingRelease(ctx, ticketIDs)
		require.NoError(t, err)

		err = service.IndexBackfill(ctx, bf)
		require.NoError(t, err)
	}

	for n := 0; n < b.N; n++ {
		for i := 0; i < 50; i++ {
			createStaleBF(fmt.Sprintf("b-%d", i), fmt.Sprintf("t1-%d", i), fmt.Sprintf("t1-%d", i+1))
		}
		err = service.CleanupBackfills(ctx)
		require.NoError(t, err)
	}
}

func TestCleanupBackfills(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	rc, err := redis.Dial("tcp", fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port")))
	require.NoError(t, err)

	bfID := "mockBackfill-1"
	ticketIDs := []string{"t1", "t2"}
	bfLastAck := "backfill_last_ack_time"
	proposedTicketIDs := "proposed_ticket_ids"
	allBackfills := "allBackfills"
	generation := int64(55)
	bf := &pb.Backfill{
		Id:         bfID,
		Generation: generation,
	}

	// ARRANGE
	err = service.CreateBackfill(ctx, bf, ticketIDs)
	require.NoError(t, err)

	// add expired but acknowledged backfill
	_, err = rc.Do("ZADD", bfLastAck, 123, bfID)
	require.NoError(t, err)

	err = service.AddTicketsToPendingRelease(ctx, ticketIDs)
	require.NoError(t, err)

	err = service.IndexBackfill(ctx, bf)
	require.NoError(t, err)

	// backfill is properly indexed
	index, err := redis.StringMap(rc.Do("HGETALL", allBackfills))
	require.NoError(t, err)
	require.Len(t, index, 1)
	require.Equal(t, strconv.Itoa(int(generation)), index[bfID])

	// ACT
	err = service.CleanupBackfills(ctx)
	require.NoError(t, err)

	// ASSERT
	// backfill must be deindexed
	index, err = redis.StringMap(rc.Do("HGETALL", allBackfills))
	require.NoError(t, err)
	require.Len(t, index, 0)

	// backfill doesn't exist anymore
	_, _, err = service.GetBackfill(ctx, bfID)
	require.Error(t, err)
	require.Equal(t, "Backfill id: mockBackfill-1 not found", status.Convert(err).Message())

	// no records in backfill sorted set left
	expiredBackfillIds, err := redis.Strings(rc.Do("ZRANGEBYSCORE", bfLastAck, 0, 200))
	require.NoError(t, err)
	require.Empty(t, expiredBackfillIds)

	// no records in tickets sorted set left
	pendingTickets, err := redis.Strings(rc.Do("ZRANGEBYSCORE", proposedTicketIDs, 0, time.Now().UnixNano()))
	require.NoError(t, err)
	require.Empty(t, pendingTickets)
}

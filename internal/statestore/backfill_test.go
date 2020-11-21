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
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	utilTesting "open-match.dev/open-match/internal/util/testing"
	"open-match.dev/open-match/pkg/pb"
)

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

	var testCases = []struct {
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

func TestUpdateBackfill(t *testing.T) {
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

	var testCases = []struct {
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
			description:     "create existing backfill, no err expected",
			backfill:        &bf,
			ticketIDs:       nil,
			expectedCode:    codes.OK,
			expectedMessage: "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			err := service.UpdateBackfill(ctx, tc.backfill, tc.ticketIDs)
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

	var testCases = []struct {
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

	err := service.CreateBackfill(ctx, &pb.Backfill{
		Id:         "mockBackfillID",
		Generation: 1,
	}, nil)
	require.NoError(t, err)

	var testCases = []struct {
		description     string
		ticketID        string
		expectedCode    codes.Code
		expectedMessage string
	}{
		{
			description:     "backfill is found and deleted",
			ticketID:        "mockBackfillID",
			expectedCode:    codes.OK,
			expectedMessage: "",
		},
		{
			description:     "empty id passed, no err expected",
			ticketID:        "",
			expectedCode:    codes.OK,
			expectedMessage: "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			errActual := service.DeleteBackfill(ctx, tc.ticketID)
			if tc.expectedCode == codes.OK {
				require.NoError(t, errActual)

				if tc.ticketID != "" {
					_, errGetTicket := service.GetTicket(ctx, tc.ticketID)
					require.Error(t, errGetTicket)
					require.Equal(t, codes.NotFound.String(), status.Convert(errGetTicket).Code().String())
				}
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

	backfills := generateBackfills(ctx, t, service, 2)
	verifyBackfills := func(service Service, backfills []*pb.Backfill) {
		ids, err := service.GetIndexedBackfills(ctx)
		require.Nil(t, err)
		require.Equal(t, len(backfills), len(ids))

		for _, tt := range backfills {
			_, ok := ids[tt.GetId()]
			require.True(t, ok)
		}
	}

	verifyBackfills(service, backfills)
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

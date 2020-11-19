package statestore

import (
	"testing"

	"github.com/stretchr/testify/require"
	utilTesting "open-match.dev/open-match/internal/util/testing"
	"open-match.dev/open-match/pkg/pb"
)

func TestNewMutex(t *testing.T) {
	cfg, closer := createRedis(t, false, "")
	defer closer()
	service := New(cfg)
	require.NotNil(t, service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	mutex := service.NewMutex("key")

	err := mutex.Lock(ctx)
	require.NoError(t, err)

	err = service.CreateBackfill(ctx, &pb.Backfill{
		Id: "222",
	}, nil)
	require.NoError(t, err)

	b, err := mutex.Unlock(ctx)
	require.NoError(t, err)
	require.True(t, b)

}

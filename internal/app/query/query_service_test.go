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
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/statestore"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
	"open-match.dev/open-match/pkg/pb"
)

func TestGetPageSize(t *testing.T) {
	testCases := []struct {
		name      string
		configure func(config.Mutable)
		expected  int
	}{
		{
			"notSet",
			func(cfg config.Mutable) {},
			1000,
		},
		{
			"set",
			func(cfg config.Mutable) {
				cfg.Set("queryPageSize", "2156")
			},
			2156,
		},
		{
			"low",
			func(cfg config.Mutable) {
				cfg.Set("queryPageSize", "9")
			},
			10,
		},
		{
			"high",
			func(cfg config.Mutable) {
				cfg.Set("queryPageSize", "10001")
			},
			10000,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cfg := viper.New()
			tt.configure(cfg)
			actual := getPageSize(cfg)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestBackfillCache(t *testing.T) {
	cfg := viper.New()
	store, closer := statestoreTesting.NewStoreServiceForTesting(t, cfg)
	defer closer()
	bfCache := &backfillCache{
		store:     store,
		backfills: make(map[string]*pb.Backfill),
	}

	t.Run("IndexedButNotInCache", func(t *testing.T) {
		bf1 := &pb.Backfill{
			Id:         "backfill-01",
			Generation: 1,
		}
		bf2 := &pb.Backfill{
			Id:         "backfill-02",
			Generation: 1,
		}
		storeAndIndex(context.Background(), store, bf1, bf2)
		bfCache.update()
		require.Equal(t, 2, len(bfCache.backfills))
	})

	t.Run("NewVersionOfBackfillIndexedButNotInCache", func(t *testing.T) {
		bf1 := &pb.Backfill{
			Id:         "backfill-01",
			Generation: 1,
		}
		bf2 := &pb.Backfill{
			Id:         "backfill-02",
			Generation: 1,
		}
		ctx := context.Background()
		storeAndIndex(ctx, store, bf1, bf2)
		bfCache.update()

		bf2v2 := &pb.Backfill{
			Id:         "backfill-02",
			Generation: 2,
		}
		store.UpdateBackfill(ctx, bf2v2, []string{})
		store.IndexBackfill(ctx, bf2v2)
		bfCache.update()
		require.Equal(t, 2, len(bfCache.backfills))
		require.Equal(t, bf2v2.Generation, bfCache.backfills[bf2v2.Id].Generation)
	})
}

func storeAndIndex(ctx context.Context, service statestore.Service, backfills ...*pb.Backfill) {
	for _, bf := range backfills {
		service.CreateBackfill(ctx, bf, []string{})
		service.IndexBackfill(ctx, bf)
	}
}

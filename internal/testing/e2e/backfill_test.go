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

package e2e

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	"open-match.dev/open-match/pkg/pb"
)

// TestCreateGetBackfill Create and Get Backfill test
func TestCreateGetBackfill(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	bf := &pb.CreateBackfillRequest{Backfill: &pb.Backfill{SearchFields: &pb.SearchFields{
		StringArgs: map[string]string{
			"search": "me",
		},
	}}}
	b1, err := om.Frontend().CreateBackfill(ctx, bf)
	require.Nil(t, err)
	require.NotNil(t, b1)

	b2, err := om.Frontend().CreateBackfill(ctx, bf)
	require.Nil(t, err)
	require.NotNil(t, b2)

	// Different IDs should be generated
	require.NotEqual(t, b1.Id, b2.Id)
	matched, err := regexp.MatchString(`[0-9a-v]{20}`, b1.GetId())
	require.True(t, matched)
	require.Nil(t, err)
	require.Equal(t, bf.Backfill.SearchFields.DoubleArgs["test-arg"], b1.SearchFields.DoubleArgs["test-arg"])
	b1.Id = b2.Id
	b1.CreateTime = b2.CreateTime

	// All fields other than CreateTime and Id fields should be equal
	require.Equal(t, b1, b2)
	require.Nil(t, err)
	actual, err := om.Frontend().GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: b1.Id})
	require.Nil(t, err)
	require.NotNil(t, actual)
	require.Equal(t, b1, actual)

	// Get Backfill which is not present - NotFound
	actual, err = om.Frontend().GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: b1.Id + "new"})
	require.NotNil(t, err)
	require.Equal(t, codes.NotFound.String(), status.Convert(err).Code().String())
	require.Nil(t, actual)
	require.Equal(t, b1, actual)
}

// TestBackfillFrontendLifecycle Create, Get and Update Backfill test
func TestBackfillFrontendLifecycle(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	bf := &pb.Backfill{SearchFields: &pb.SearchFields{
		StringArgs: map[string]string{
			"search": "me",
		},
	},
	}
	createdBf, err := om.Frontend().CreateBackfill(ctx, &pb.CreateBackfillRequest{Backfill: bf})
	require.NoError(t, err)
	require.Equal(t, int64(1), createdBf.Generation)

	createdBf.SearchFields.StringArgs["key"] = "val"

	orig := &anypb.Any{Value: []byte("test")}
	val, err := ptypes.MarshalAny(orig)
	require.NoError(t, err)

	// Create a different Backfill, but with the same ID
	// Pass different time
	bf2 := &pb.Backfill{
		CreateTime: ptypes.TimestampNow(),
		Id:         createdBf.Id,
		SearchFields: &pb.SearchFields{
			StringArgs: map[string]string{
				"key":    "val",
				"search": "me",
			},
		},
		Generation: 42,
		Extensions: map[string]*any.Any{"key": val},
	}
	updatedBf, err := om.Frontend().UpdateBackfill(ctx, &pb.UpdateBackfillRequest{Backfill: bf2})
	require.NoError(t, err)
	require.Equal(t, int64(2), updatedBf.Generation)

	// No changes to CreateTime
	require.Equal(t, createdBf.CreateTime.GetNanos(), updatedBf.CreateTime.GetNanos())

	get, err := om.Frontend().GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: createdBf.Id})
	require.NoError(t, err)
	require.Equal(t, bf2.SearchFields.StringArgs, get.SearchFields.StringArgs)

	unpacked := &anypb.Any{}
	err = ptypes.UnmarshalAny(get.Extensions["key"], unpacked)
	require.NoError(t, err)

	require.Equal(t, unpacked.Value, orig.Value)
	_, err = om.Frontend().DeleteBackfill(ctx, &pb.DeleteBackfillRequest{BackfillId: createdBf.Id})
	require.NoError(t, err)

	get, err = om.Frontend().GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: createdBf.Id})
	require.Error(t, err, fmt.Sprintf("Backfill id: %s not found", bf.Id))
	require.Nil(t, get)
}

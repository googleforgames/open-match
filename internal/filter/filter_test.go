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

package filter

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"open-match.dev/open-match/internal/filter/testcases"
	"open-match.dev/open-match/pkg/pb"
)

func TestMeetsCriteria(t *testing.T) {
	testInclusion := func(t *testing.T, pool *pb.Pool, entity filteredEntity) {
		pf, err := NewPoolFilter(pool)

		require.NoError(t, err)
		require.NotNil(t, pf)

		if !pf.In(entity) {
			t.Error("entity should be included in the pool")
		}
	}

	for _, tc := range testcases.IncludedTestCases() {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			testInclusion(t, tc.Pool, &pb.Ticket{
				SearchFields: tc.SearchFields,
				CreateTime:   timestamppb.Now(),
			})
			testInclusion(t, tc.Pool, &pb.Backfill{
				SearchFields: tc.SearchFields,
				CreateTime:   timestamppb.Now(),
			})
		})
	}

	testExclusion := func(t *testing.T, pool *pb.Pool, entity filteredEntity) {
		pf, err := NewPoolFilter(pool)

		require.NoError(t, err)
		require.NotNil(t, pf)

		if pf.In(entity) {
			t.Error("ticket should be excluded from the pool")
		}
	}

	for _, tc := range testcases.ExcludedTestCases() {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			testExclusion(t, tc.Pool, &pb.Ticket{
				SearchFields: tc.SearchFields,
				CreateTime:   timestamppb.Now(),
			})
			testExclusion(t, tc.Pool, &pb.Backfill{
				SearchFields: tc.SearchFields,
				CreateTime:   timestamppb.Now(),
			})
		})
	}
}

func TestValidPoolFilter(t *testing.T) {
	for _, tc := range []struct {
		name string
		pool *pb.Pool
		code codes.Code
		msg  string
	}{
		{
			"invalid create before",
			&pb.Pool{
				CreatedBefore: &timestamppb.Timestamp{Nanos: -1},
			},
			codes.InvalidArgument,
			".invalid created_before value",
		},
		{
			"invalid create after",
			&pb.Pool{
				CreatedAfter: &timestamppb.Timestamp{Nanos: -1},
			},
			codes.InvalidArgument,
			".invalid created_after value",
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pf, err := NewPoolFilter(tc.pool)

			require.Error(t, err)
			require.Nil(t, pf)

			s := status.Convert(err)
			require.Equal(t, tc.code, s.Code())
			require.Equal(t, tc.msg, s.Message())
		})
	}
}

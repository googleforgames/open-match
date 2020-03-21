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

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/filter/testcases"
	"open-match.dev/open-match/pkg/pb"
)

func TestMeetsCriteria(t *testing.T) {
	for _, tc := range testcases.IncludedTestCases() {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			pf, err := NewPoolFilter(tc.Pool)
			if err != nil {
				t.Error("pool should be valid")
			}
			tc.Ticket.CreateTime = ptypes.TimestampNow()
			if !pf.In(tc.Ticket) {
				t.Error("ticket should be included in the pool")
			}
		})
	}

	for _, tc := range testcases.ExcludedTestCases() {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			pf, err := NewPoolFilter(tc.Pool)
			if err != nil {
				t.Error("pool should be valid")
			}
			tc.Ticket.CreateTime = ptypes.TimestampNow()
			if pf.In(tc.Ticket) {
				t.Error("ticket should be excluded from the pool")
			}
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
				CreatedBefore: &timestamp.Timestamp{Nanos: -1},
			},
			codes.InvalidArgument,
			".invalid created_before value",
		},
		{
			"invalid create after",
			&pb.Pool{
				CreatedAfter: &timestamp.Timestamp{Nanos: -1},
			},
			codes.InvalidArgument,
			".invalid created_after value",
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pf, err := NewPoolFilter(tc.pool)
			assert.Nil(t, pf)
			s := status.Convert(err)
			assert.Equal(t, tc.code, s.Code())
			assert.Equal(t, tc.msg, s.Message())
		})
	}
}

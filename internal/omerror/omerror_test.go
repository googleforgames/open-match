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

package omerror

import (
	"context"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestProtoFromErr(t *testing.T) {
	tests := []struct {
		err  error
		want *spb.Status
	}{
		{
			nil,
			&spb.Status{Code: int32(codes.OK)},
		},
		{
			context.Canceled,
			&spb.Status{Code: int32(codes.Canceled), Message: "context canceled"},
		},
		{
			context.DeadlineExceeded,
			&spb.Status{Code: int32(codes.DeadlineExceeded), Message: "context deadline exceeded"},
		},
		{
			fmt.Errorf("monkeys with no hats"),
			&spb.Status{Code: int32(codes.Unknown), Message: "monkeys with no hats"},
		},
		{
			status.Errorf(codes.Internal, "even the lemurs have no hats"),
			&spb.Status{Code: int32(codes.Internal), Message: "even the lemurs have no hats"},
		},
	}

	for _, tc := range tests {
		require.Equal(t, tc.want, ProtoFromErr(tc.err))
	}
}

func TestWaitOnErrors(t *testing.T) {
	errA := fmt.Errorf("the fish have the hats")
	errB := fmt.Errorf("who gave the fish hats")

	tests := []struct {
		err    error
		fs     []func() error
		logged bool
		log    string
	}{
		{
			nil, []func() error{}, false, "",
		},
		{
			errA,
			[]func() error{
				func() error {
					return errA
				},
			},
			false, "",
		},
		{
			nil,
			[]func() error{
				func() error {
					return nil
				},
			},
			false, "",
		},
		{
			errB,
			[]func() error{
				func() error {
					return errB
				},
				func() error {
					return errB
				},
			},
			true, "Multiple errors occurred in parallel execution. This error is suppressed by the error returned.",
		},
	}

	for _, tc := range tests {
		logger, hook := test.NewNullLogger()
		wait := WaitOnErrors(logrus.NewEntry(logger), tc.fs...)

		require.Equal(t, tc.err, wait())

		if tc.logged {
			require.Equal(t, 1, len(hook.Entries))
			require.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
			require.Equal(t, tc.log, hook.LastEntry().Message)
		} else {
			require.Nil(t, hook.LastEntry())
		}
	}

	_ = errB /////////////////////////////////////////////////////////////////////////////
}

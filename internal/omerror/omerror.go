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

	"github.com/sirupsen/logrus"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ProtoFromErr converts an error into a grpc status.  It differs from
// google.golang.org/grpc/status in that it will return an OK code on nil, and
// returns the proper codes for context cancelation and deadline exceeded.
func ProtoFromErr(err error) *spb.Status {
	switch err {
	case nil:
		return &spb.Status{Code: int32(codes.OK)}
	case context.DeadlineExceeded:
		fallthrough
	case context.Canceled:
		return status.FromContextError(err).Proto()
	default:
		return status.Convert(err).Proto()
	}
}

// WaitFunc will wait until all called functions return.  WaitFunc returns the
// first error returned, otherwise it returns nil.
type WaitFunc func() error

// WaitOnErrors immediately starts a new go routine for each function passed it.
// It returns a WaitFunc. Any additional errors not returned are instead logged.
func WaitOnErrors(logger *logrus.Entry, fs ...func() error) WaitFunc {
	errors := make(chan error, len(fs))
	for _, f := range fs {
		go func(f func() error) {
			errors <- f()
		}(f)
	}

	return func() error {
		var first error
		for range fs {
			err := <-errors
			if first == nil {
				first = err
			} else {
				if err != nil {
					logger.WithError(err).Warning("Multiple errors occurred in parallel execution. This error is suppressed by the error returned.")
				}
			}
		}
		return first
	}
}

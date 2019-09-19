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

// Package util provides utilities for net.Listener.
package util

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/metadata"
)

const (
	metadataNameSynchronizerContextID = "synchronizer-context-id"
)

// AppendSynchronizerContextID adds the synchronizer context id to a request context metadata.
func AppendSynchronizerContextID(ctx context.Context, id string) (context.Context, error) {
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		values := getSynchronizerContextIDFromMetadata(md)
		if len(values) == 1 {
			if values[0] != id {
				return ctx, errors.New("request already has a different synchronizer context id")
			}

			return ctx, nil
		}

		if len(values) > 1 {
			return ctx, errors.New("context has more than 1 synchronizer context ids")
		}
	}

	return metadata.AppendToOutgoingContext(ctx, metadataNameSynchronizerContextID, id), nil
}

// GetSynchronizerContextID returns the synchronizer context id from the context metadata.
func GetSynchronizerContextID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	values := getSynchronizerContextIDFromMetadata(md)
	if len(values) == 1 {
		return values[0]
	}

	return ""
}

func getSynchronizerContextIDFromMetadata(md metadata.MD) []string {
	values := md.Get(metadataNameSynchronizerContextID)
	if len(values) > 0 {
		return values
	}

	return []string{}
}

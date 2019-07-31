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

// Package testing provides utility methods for testing.
package testing

import (
	"context"
	"testing"
)

type contextTestKey string

// NewContext returns a context appropriate for calling a gRPC service that
// is not hosted on a cluster or in Minimatch, (ie: tests not in test/e2e/).
// For those tests use OM.Context() method.
func NewContext(t *testing.T) context.Context {
	return context.WithValue(context.Background(), contextTestKey("testing.T"), t)
}

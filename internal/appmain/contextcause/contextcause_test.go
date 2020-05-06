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

package contextcause

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var testErr = errors.New("testErr")

func TestCauseOverride(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	ctx, cancel := WithCancelCause(parent)

	select {
	case <-ctx.Done():
		t.FailNow()
	default:
	}

	cancel(testErr)
	<-ctx.Done()
	require.Equal(t, testErr, ctx.Err())

	cancel(errors.New("second error"))
	require.Equal(t, testErr, ctx.Err())

	cancelParent()
	require.Equal(t, testErr, ctx.Err())
}

func TestParentCanceledFirst(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	ctx, cancel := WithCancelCause(parent)

	select {
	case <-ctx.Done():
		t.FailNow()
	default:
	}

	cancelParent()
	<-ctx.Done()
	require.Equal(t, context.Canceled, ctx.Err())

	cancel(testErr)
	require.Equal(t, context.Canceled, ctx.Err())
}

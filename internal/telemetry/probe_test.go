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

package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func angryHealthCheck(context.Context) error {
	return fmt.Errorf("I'm angry")
}

func happyHealthCheck(context.Context) error {
	return nil
}

func TestAlwaysReadyHealthCheck(t *testing.T) {
	assertHealthCheck(t, NewAlwaysReadyHealthCheck(), "")
}

func TestHealthCheck(t *testing.T) {
	var testCases = []struct {
		name         string
		healthChecks []func(context.Context) error
		errorString  string
	}{
		{"Empty", []func(context.Context) error{}, ""},
		{"happyHealthCheck", []func(context.Context) error{happyHealthCheck}, ""},
		{"angryHealthCheck", []func(context.Context) error{angryHealthCheck}, "I'm angry"},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assertHealthCheck(t, NewHealthCheck(tc.healthChecks), tc.errorString)
		})
	}
}

func assertHealthCheck(t *testing.T, hc http.Handler, errorString string) {
	require := require.New(t)
	sp, ok := hc.(*statefulProbe)
	require.True(ok)
	require.Equal(healthStateFirstProbe, atomic.LoadInt32(sp.healthState))

	hcFunc := func(w http.ResponseWriter, r *http.Request) {
		hc.ServeHTTP(w, r)
	}
	require.HTTPSuccess(hcFunc, http.MethodGet, "/", url.Values{}, "ok")
	// A readiness probe has not happened yet so it's still in "first state"
	require.Equal(healthStateFirstProbe, atomic.LoadInt32(sp.healthState))
	if errorString == "" {
		require.HTTPSuccess(hcFunc, http.MethodGet, "/", url.Values{"readiness": []string{"true"}}, "ok")
		require.Equal(healthStateHealthy, atomic.LoadInt32(sp.healthState))
	} else {
		require.HTTPError(hcFunc, http.MethodGet, "/", url.Values{"readiness": []string{"true"}}, errorString)
		require.Equal(healthStateUnhealthy, atomic.LoadInt32(sp.healthState))
	}
}

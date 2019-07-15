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
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
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
		t.Run(tc.name, func(t *testing.T) {
			assertHealthCheck(t, NewHealthCheck(tc.healthChecks), tc.errorString)
		})
	}
}

func assertHealthCheck(t *testing.T, hc http.Handler, errorString string) {
	assert := assert.New(t)
	sp, ok := hc.(*statefulProbe)
	assert.True(ok)
	assert.Equal(healthStateFirstProbe, atomic.LoadInt32(sp.healthState))

	hcFunc := func(w http.ResponseWriter, r *http.Request) {
		hc.ServeHTTP(w, r)
	}
	assert.HTTPSuccess(hcFunc, http.MethodGet, "/", url.Values{}, "ok")
	// A readiness probe has not happened yet so it's still in "first state"
	assert.Equal(healthStateFirstProbe, atomic.LoadInt32(sp.healthState))
	if errorString == "" {
		assert.HTTPSuccess(hcFunc, http.MethodGet, "/", url.Values{"readiness": []string{"true"}}, "ok")
		assert.Equal(healthStateHealthy, atomic.LoadInt32(sp.healthState))
	} else {
		assert.HTTPError(hcFunc, http.MethodGet, "/", url.Values{"readiness": []string{"true"}}, errorString)
		assert.Equal(healthStateUnhealthy, atomic.LoadInt32(sp.healthState))
	}
}

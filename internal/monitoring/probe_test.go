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

package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func angryHealthCheck(context.Context) error {
	return fmt.Errorf("I'm angry")
}

func happyHealthCheck(context.Context) error {
	return nil
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
			assert := assert.New(t)
			hc := NewHealthProbe(tc.healthChecks)
			assert.HTTPSuccess(hc, http.MethodGet, "/", url.Values{}, "ok")
			if tc.errorString == "" {
				assert.HTTPSuccess(hc, http.MethodGet, "/?readiness=true", url.Values{}, "ok")
			} else {
				assert.HTTPError(hc, http.MethodGet, "/", url.Values{"readiness": []string{"true"}}, tc.errorString)
			}
		})
	}
}

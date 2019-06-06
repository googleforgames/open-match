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
)

// NewHealthProbe creates an HTTP handler for Kubernetes liveness and readiness checks.
func NewHealthProbe(probes []func(context.Context) error) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if len(req.URL.Query()) > 0 {
			// Readiness probe are triggered if there's a query (ie "?" in the url).
			// If so then scan all the probes.
			for _, probe := range probes {
				err := probe(req.Context())
				if err != nil {
					http.Error(w, err.Error(), http.StatusServiceUnavailable)
					return
				}
			}
		}
		fmt.Fprintf(w, "ok")
	}
}

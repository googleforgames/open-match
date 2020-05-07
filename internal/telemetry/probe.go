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
	"sync/atomic"
)

const (
	// HealthCheckEndpoint is the endpoint for Kubernetes health probes.
	// See: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/
	HealthCheckEndpoint   = "/healthz"
	healthStateFirstProbe = int32(0)
	healthStateHealthy    = int32(1)
	healthStateUnhealthy  = int32(2)
)

type statefulProbe struct {
	healthState *int32
	probes      []func(context.Context) error
}

// ServeHTTP serves the health check endpoint to tell Kubernetes that the service is healthy.
// This class will print logs based on the health status.
func (sp *statefulProbe) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if len(req.URL.Query()) > 0 {
		// Readiness probe are triggered if there's a query (ie "?" in the url).
		// If so then scan all the probes.
		for _, probe := range sp.probes {
			err := probe(req.Context())
			if err != nil {
				old := atomic.SwapInt32(sp.healthState, healthStateUnhealthy)
				if old == healthStateUnhealthy {
					logger.WithError(err).Warningf("%s health check continues to fail. The server is at risk of termination.", HealthCheckEndpoint)
				} else {
					logger.WithError(err).Warningf("%s health check failed. The server will terminate if this continues to happen.", HealthCheckEndpoint)
				}
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
		}

		old := atomic.SwapInt32(sp.healthState, healthStateHealthy)
		if old == healthStateUnhealthy {
			logger.Infof("%s is healthy again.", HealthCheckEndpoint)
		} else if old == healthStateFirstProbe {
			logger.Infof("%s is reporting healthy.", HealthCheckEndpoint)
		}
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "ok")
}

// NewAlwaysReadyHealthCheck indicates that the service is always healthy. Used for static HTTP servers.
func NewAlwaysReadyHealthCheck() http.Handler {
	return NewHealthCheck([]func(context.Context) error{})
}

// NewHealthCheck creates an HTTP handler for Kubernetes liveness and readiness checks.
func NewHealthCheck(probes []func(context.Context) error) http.Handler {
	return &statefulProbe{
		healthState: new(int32),
		probes:      probes,
	}
}

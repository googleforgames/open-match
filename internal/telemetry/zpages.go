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
	"net/http"
	"net/http/pprof"

	"github.com/sirupsen/logrus"
	"go.opencensus.io/zpages"
)

const (
	debugEndpoint                    = "/debug"
	configNameTelemetryZpagesEnabled = "telemetry.zpages.enable"
)

func bindZpages(p Params, b Bindings) error {
	if !p.Config().GetBool(configNameTelemetryZpagesEnabled) {
		logger.Info("zPages: Disabled")
		return nil
	}
	logger.WithFields(logrus.Fields{
		"endpoint": debugEndpoint,
	}).Info("zPages: ENABLED")

	mux := http.NewServeMux()

	zpages.Handle(mux, debugEndpoint)

	mux.HandleFunc(debugEndpoint+"/pprof/", pprof.Index)
	mux.HandleFunc(debugEndpoint+"/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc(debugEndpoint+"/pprof/profile", pprof.Profile)
	mux.HandleFunc(debugEndpoint+"/pprof/symbol", pprof.Symbol)
	mux.HandleFunc(debugEndpoint+"/pprof/trace", pprof.Trace)

	b.TelemetryHandle(debugEndpoint, mux)

	return nil
}

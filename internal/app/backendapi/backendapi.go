// Copyright 2018 Google LLC
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
package backendapi

import (
	"github.com/GoogleCloudPlatform/open-match/internal/app/backendapi/apisrv"
	"github.com/GoogleCloudPlatform/open-match/internal/metrics"

	"github.com/GoogleCloudPlatform/open-match/internal/serving"

	"github.com/sirupsen/logrus"
)

// CreateServerParams creates the configuration and prepares the binding for serving handler.
func CreateServerParams() *serving.ServerParams {
	// Add a hook to the logger to auto-count log lines for metrics output thru OpenCensus
	logrus.AddHook(metrics.NewHook(apisrv.BeLogLines, apisrv.KeySeverity))

	return &serving.ServerParams{
		BaseLogFields: logrus.Fields{
			"app":       "openmatch",
			"component": "backend",
		},
		ServicePortConfigName: "api.backend.port",
		ProxyPortConfigName:   "api.backend.proxyport",
		CustomMeasureViews:    apisrv.DefaultBackendAPIViews,
		Bindings:              []serving.BindingFunc{apisrv.Bind},
	}
}

// RunApplication is a hook for the main() method in the main executable.
func RunApplication() {
	params := CreateServerParams()
	serving.MustServeForever(params)
}

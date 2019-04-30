/*
This application handles all the startup and connection scaffolding for
running a gRPC server serving the APIService as defined in
${OM_ROOT}/internal/pb/frontend.pb.go

All the actual important bits are in the API Server source code: apisrv/apisrv.go

Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package frontendapi

import (
	"github.com/GoogleCloudPlatform/open-match/internal/app/frontendapi/apisrv"
	"github.com/GoogleCloudPlatform/open-match/internal/metrics"
	"github.com/GoogleCloudPlatform/open-match/internal/serving"

	"github.com/sirupsen/logrus"
)

// CreateServerParams creates the configuration and prepares the binding for serving handler.
func CreateServerParams() *serving.ServerParams {
	// Add a hook to the logger to auto-count log lines for metrics output thru OpenCensus
	logrus.AddHook(metrics.NewHook(apisrv.FeLogLines, apisrv.KeySeverity))

	return &serving.ServerParams{
		BaseLogFields: logrus.Fields{
			"app":       "openmatch",
			"component": "frontend",
		},
		ServicePortConfigName: "api.frontend.port",
		ProxyPortConfigName:   "api.frontend.proxyport",
		CustomMeasureViews:    apisrv.DefaultFrontendAPIViews,
		Bindings:              []serving.BindingFunc{apisrv.Bind},
	}
}

// RunApplication is a hook for the main() method in the main executable.
func RunApplication() {
	params := CreateServerParams()
	serving.MustServeForever(params)
}

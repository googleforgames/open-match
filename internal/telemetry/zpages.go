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

	"github.com/sirupsen/logrus"
	"go.opencensus.io/zpages"
	"open-match.dev/open-match/internal/config"
)

func bindZpages(mux *http.ServeMux, cfg config.View) {
	if !cfg.GetBool("telemetry.zpages.enable") {
		logger.Info("zPages: Disabled")
		return
	}
	endpoint := "/debug"
	zpages.Handle(mux, endpoint)

	logger.WithFields(logrus.Fields{
		"endpoint": endpoint,
	}).Info("zPages: ENABLED")
}

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

package minimatch

import (
	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/app/backend"
	"open-match.dev/open-match/internal/app/frontend"
	"open-match.dev/open-match/internal/app/mmlogic"
	"open-match.dev/open-match/internal/app/synchronizer"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "minimatch",
	})
)

// RunApplication creates a server.
func RunApplication() {
	cfg, err := config.Read()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}
	p, err := rpc.NewServerParamsFromConfig(cfg, "api.frontend")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot construct server.")
	}

	if err := BindService(p, cfg); err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot bind server.")
	}

	rpc.MustServeForever(p)
}

// BindService creates the minimatch service to the server Params.
func BindService(p *rpc.ServerParams, cfg config.View) error {
	if err := backend.BindService(p, cfg); err != nil {
		return err
	}

	if err := frontend.BindService(p, cfg); err != nil {
		return err
	}

	if err := mmlogic.BindService(p, cfg); err != nil {
		return err
	}

	if err := synchronizer.BindService(p, cfg); err != nil {
		return err
	}

	return nil
}

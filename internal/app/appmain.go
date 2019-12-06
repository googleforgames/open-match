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

// Package app contains the common application initialization code for Open Match servers.
package app

import (
	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/logging"
	"open-match.dev/open-match/internal/rpc"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.main",
	})
)

// RunApplication creates a server.
func RunApplication(serverName string, getCfg func() (config.View, error), bindService func(*rpc.ServerParams, config.View) error) {
	cfg, err := getCfg()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}
	logging.ConfigureLogging(cfg)
	p, err := rpc.NewServerParamsFromConfig(cfg, "api."+serverName)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot construct server.")
	}

	if err := bindService(p, cfg); err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("failed to bind %s service.", serverName)
	}

	rpc.MustServeForever(p)
}

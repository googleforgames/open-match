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

package frontend

import (
	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/examples/scale/tickets"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/logging"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "scale.frontend",
	})
)

// Run triggers execution of the scale frontend component that creates
// tickets at scale in Open Match.
func Run() {
	cfg, err := config.Read()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("cannot read configuration.")
	}

	logging.ConfigureLogging(cfg)

	// TODO: This is a placeholder - add the actual implementation.
	concurrent := cfg.GetInt("testConfig.concurrent-creates")
	for i := 0; i <= concurrent; i++ {
		_ = tickets.Ticket(cfg)
	}
}

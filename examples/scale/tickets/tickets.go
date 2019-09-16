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

package tickets

import (
	"open-match.dev/open-match/internal/config"

	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/pkg/pb"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "scale-frontend.tickets",
	})
)

// Ticket generates a ticket based on the config for scale testing
func Ticket(cfg config.View) *pb.Ticket {
	// TODO: Add implementation for generating a fake ticket.
	_ = cfg
	logger.Info("Tickets.Ticket() invoked")
	return &pb.Ticket{}
}

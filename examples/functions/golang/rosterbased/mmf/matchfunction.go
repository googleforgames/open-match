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

// Package mmf provides a sample match function that uses the GRPC harness to set up 1v1 matches.
// This sample is a reference to demonstrate the usage of the GRPC harness and should only be used as
// a starting point for your match function. You will need to modify the
// matchmaking logic in this function based on your game's requirements.
package mmf

import (
	"github.com/sirupsen/logrus"
	mmfHarness "open-match.dev/open-match/pkg/harness/function/golang"
	"open-match.dev/open-match/pkg/pb"
)

var (
	matchName = "roster-based-matchfunction"
	logger    = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "mmf.rosterbased",
	})
)

// MakeMatches is where your custom matchmaking logic lives.
func MakeMatches(p *mmfHarness.MatchFunctionParams) ([]*pb.Match, error) {
	logger.Infof("MakeMatches called for Profile:%v", p.ProfileName)
	return []*pb.Match{}, nil
}

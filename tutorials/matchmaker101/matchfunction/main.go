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

// Package main defines a sample match function that uses the GRPC harness to set up
// the match making function as a service. This sample is a reference
// to demonstrate the usage of the GRPC harness and should only be used as
// a starting point for your match function. You will need to modify the
// matchmaking logic in this function based on your game's requirements.
package main

import (
	"open-match.dev/open-match/tutorials/matchmaker101/matchfunction/mmf"
)

// This tutorial implenents a basic Match Function that is hosted in the below
// configured port. You can also configure the Open Match QueryService endpoint
// with which the Match Function communicates to query the Tickets.

const (
	queryServiceAddress = "open-match-query.open-match.svc.cluster.local:50503" // Address of the QueryService endpoint.
	serverPort          = 50502                                                 // The port for hosting the Match Function.
)

func main() {
	mmf.Start(queryServiceAddress, serverPort)
}

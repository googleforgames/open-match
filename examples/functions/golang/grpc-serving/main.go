/*
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

package main

import (
	"context"

	harness "github.com/GoogleCloudPlatform/open-match/internal/harness/matchfunction/golang"
	"github.com/GoogleCloudPlatform/open-match/internal/pb"
)

func main() {
	// Invoke the harness to setup a GRPC service that handles requests to run the
	// match function. The harness itself queries open match for player pools for
	// the specified request and passes the pools to the match function to generate
	// proposals.
	harness.ServeMatchFunction(&harness.HarnessParams{
		FunctionName:          "simple-matchfunction",
		ServicePortConfigName: "api.functions.port",
		ProxyPortConfigName:   "api.functions.proxyport",
		Func:                  makeMatches,
	})
}

func makeMatches(ctx context.Context, profile string, rosters []*pb.Roster, pools []*pb.PlayerPool) (string, []*pb.Roster, error) {
	// TODO - Add logic to generate match proposals.
	return "", nil, nil
}

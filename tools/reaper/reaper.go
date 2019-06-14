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

// Package main is the main for the reaper tool to cleanup leaked clusters in Open Match CI.
package main

import (
	"flag"
	"time"

	reaperInternal "open-match.dev/open-match/tools/reaper/internal"
)

var (
	projectIDFlag = flag.String("project", "open-match-build", "Target Project ID to scan for leaked clusters.")
	ageFlag       = flag.Duration("age", time.Hour*1, "Age of a cluster before it's a candidate for deletion.")
	labelFlag     = flag.String("label", "open-match-ci", "Required label that must be on the cluster for it to be considered for reaping.")
	locationFlag  = flag.String("location", "us-west1-a", "Location of the cluster")
)

func main() {
	flag.Parse()
	reaperInternal.ReapCluster(&reaperInternal.Params{
		Age:       *ageFlag,
		Label:     *labelFlag,
		ProjectID: *projectIDFlag,
		Location:  *locationFlag,
	})
}

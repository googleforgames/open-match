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
	"fmt"
	"log"
	"net"
	reaperInternal "open-match.dev/open-match/tools/reaper/internal"
	"os"
	"strconv"
	"time"
)

var (
	projectIDFlag = flag.String("project", "open-match-build", "Target Project ID to scan for leaked clusters.")
	ageFlag       = flag.Duration("age", time.Hour*1, "Age of a cluster before it's a candidate for deletion.")
	labelFlag     = flag.String("label", "open-match-ci", "Required label that must be on the cluster for it to be considered for reaping.")
	locationFlag  = flag.String("location", "us-west1-a", "Location of the cluster")
	portFlag      = flag.Int("port", 0, "HTTP Port to serve from.")
)

func main() {
	flag.Parse()
	port := 0
	portStr := os.Getenv("PORT")
	if portStr != "" {
		var err error
		port, err = strconv.Atoi(portStr)
		if err != nil {
			log.Printf("Not serving via $PORT variable, %s, because it's not an integer, %s.", portStr, err)
		}
	} else {
		port = *portFlag
	}
	if port > 0 {
		addr := fmt.Sprintf(":%d", port)
		conn, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("cannot serve on %d, %s", port, err)
		}
		if err := reaperInternal.Serve(conn, flagToParams()); err != nil {
			log.Fatal(err)
		}
	} else {
		if resp, err := reaperInternal.ReapCluster(flagToParams()); err != nil {
			log.Fatal(err)
		} else {
			log.Print(resp)
		}
	}
}

func flagToParams() *reaperInternal.Params {
	return &reaperInternal.Params{
		Age:       *ageFlag,
		Label:     *labelFlag,
		ProjectID: *projectIDFlag,
		Location:  *locationFlag,
	}
}

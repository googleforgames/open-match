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
	"os"
	"strconv"
	"time"

	reaperInternal "open-match.dev/open-match/tools/reaper/internal"
)

var (
	ageFlag  = flag.Duration("age", time.Minute*30, "Age of a namespace before it's a candidate for deletion.")
	portFlag = flag.Int("port", 0, "HTTP Port to serve from.")
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
		if err := reaperInternal.ReapNamespaces(flagToParams()); err != nil {
			log.Fatal(err)
		} else {
			log.Print("OK")
		}
	}
}

func flagToParams() *reaperInternal.Params {
	return &reaperInternal.Params{
		Age: *ageFlag,
	}
}

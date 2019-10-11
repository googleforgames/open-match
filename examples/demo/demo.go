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

// Package demo contains the core startup code for running a demo.
package demo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/net/websocket"
	"open-match.dev/open-match/examples/demo/bytesub"
	"open-match.dev/open-match/examples/demo/components"
	"open-match.dev/open-match/examples/demo/updater"
	"open-match.dev/open-match/internal/telemetry"
)

// Run starts the provided components, and hosts a webserver for observing the
// output of those components.
func Run(comps map[string]func(*components.DemoShared)) {
	log.Print("Initializing Server")

	fileServe := http.FileServer(http.Dir("/app/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fileServe))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		fileServe.ServeHTTP(w, r)
	})

	http.Handle(telemetry.HealthCheckEndpoint, telemetry.NewAlwaysReadyHealthCheck())

	bs := bytesub.New()
	u := updater.New(context.Background(), func(b []byte) {
		var out bytes.Buffer
		err := json.Indent(&out, b, "", "  ")
		if err == nil {
			bs.AnnounceLatest(out.Bytes())
		} else {
			bs.AnnounceLatest(b)
		}
	})

	http.Handle("/connect", websocket.Handler(func(ws *websocket.Conn) {
		bs.Subscribe(ws.Request().Context(), ws)
	}))

	log.Print("Starting Server")

	for name, f := range comps {
		go f(&components.DemoShared{
			Ctx:    context.Background(),
			Update: u.ForField(name),
		})
	}

	address := fmt.Sprintf(":%d", 51507)
	err := http.ListenAndServe(address, nil)
	log.Printf("HTTP server closed: %s", err.Error())
}

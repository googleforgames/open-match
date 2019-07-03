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

package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"
	"open-match.dev/open-match/examples/demo/bytesub"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/logging"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "examples.demo",
	})
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}
	logging.ConfigureLogging(cfg)

	logger.Info("Initializing Server")

	fileServe := http.FileServer(http.Dir("/app/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fileServe))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		fileServe.ServeHTTP(w, r)
	})

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	})

	bs := bytesub.New()
	go func() {
		i := 0
		for range time.Tick(time.Second) {
			bs.AnnounceLatest([]byte(fmt.Sprintf("Uptime: %d", i)))
			i++
		}
	}()

	http.Handle("/connect", websocket.Handler(func(ws *websocket.Conn) {
		bs.Subscribe(ws.Request().Context(), ws)
	}))

	logger.Info("Starting Server")

	// TODO: Other services read their port from the common config map, how should
	// this be choosing the ports it exposes?
	err = http.ListenAndServe(":51507", nil)
	logger.WithError(err).Fatal("Http ListenAndServe failed.")
}

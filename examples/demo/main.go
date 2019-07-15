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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"
	"net/http"
	"open-match.dev/open-match/examples/demo/bytesub"
	"open-match.dev/open-match/examples/demo/components"
	"open-match.dev/open-match/examples/demo/components/clients"
	"open-match.dev/open-match/examples/demo/components/director"
	"open-match.dev/open-match/examples/demo/components/uptime"
	"open-match.dev/open-match/examples/demo/updater"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/logging"
	"open-match.dev/open-match/internal/telemetry"
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

	logger.Info("Starting Server")

	go startComponents(cfg, u)

	address := fmt.Sprintf(":%d", cfg.GetInt("api.demo.httpport"))
	err = http.ListenAndServe(address, nil)
	logger.WithError(err).Warning("HTTP server closed.")
}

func startComponents(cfg config.View, u *updater.Updater) {
	for name, f := range map[string]func(*components.DemoShared){
		"uptime":   uptime.Run,
		"clients":  clients.Run,
		"director": director.Run,
	} {
		go f(&components.DemoShared{
			Ctx:    context.Background(),
			Cfg:    cfg,
			Update: u.ForField(name),
		})
	}
}

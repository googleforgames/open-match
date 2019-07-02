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

// Package swaggerui is a simple webserver for hosting Open Match Swagger UI.
package swaggerui

import (
	"context"
	"os"

	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/monitoring"
)

var (
	swaggeruiLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "swaggerui",
	})
)

// RunApplication is the main for swagger ui http reverse proxy.
func RunApplication() {
	cfg, err := config.Read()
	if err != nil {
		swaggeruiLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}

	serve(cfg)
}

func serve(cfg config.View) {
	mux := &http.ServeMux{}
	monitoring.Setup(mux, cfg)
	port := cfg.GetInt("api.swaggerui.httpport")
	baseDir, err := os.Getwd()
	if err != nil {
		swaggeruiLogger.WithError(err).Fatal("cannot get current working directory")
	}
	directory := filepath.Join(baseDir, "static")

	_, err = os.Stat(directory)
	if err != nil {
		swaggeruiLogger.WithError(err).Fatalf("Cannot access directory %s", directory)
	}

	mux.Handle("/", http.FileServer(http.Dir(directory)))
	mux.HandleFunc("/healthz", monitoring.NewHealthProbe([]func(context.Context) error{}))
	bindHandler(mux, cfg, "/v1/frontend/", "frontend")
	bindHandler(mux, cfg, "/v1/backend/", "backend")
	bindHandler(mux, cfg, "/v1/mmlogic/", "mmlogic")
	bindHandler(mux, cfg, "/v1/synchronizer/", "synchronizer")
	bindHandler(mux, cfg, "/v1/evaluator/", "evaluator")
	bindHandler(mux, cfg, "/v1/matchfunction/", "functions")
	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	swaggeruiLogger.Infof("SwaggerUI for Open Match")
	swaggeruiLogger.Infof("Serving static directory: %s", directory)
	swaggeruiLogger.Infof("Serving on %s", addr)
	swaggeruiLogger.Fatal(srv.ListenAndServe())
}

func bindHandler(mux *http.ServeMux, cfg config.View, path string, service string) {
	hostname := cfg.GetString("api." + service + ".hostname")
	httpport := cfg.GetString("api." + service + ".httpport")
	endpoint := fmt.Sprintf("http://%s:%s/", hostname, httpport)
	swaggeruiLogger.Infof("Registering reverse proxy %s -> %s", path, endpoint)
	mux.Handle(path, overlayURLProxy(mustURLParse(endpoint)))
}

// Reference implementation: https://golang.org/src/net/http/httputil/reverseproxy.go?s=3330:3391#L98
func overlayURLProxy(target *url.URL) *httputil.ReverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
		swaggeruiLogger.Debugf("URL: %s", req.URL)
	}
	return &httputil.ReverseProxy{Director: director}
}

func mustURLParse(u string) *url.URL {
	url, err := url.Parse(u)
	if err != nil {
		swaggeruiLogger.Fatalf("failed to parse URL %s, %v", u, err)
	}
	return url
}

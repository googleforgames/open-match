// Copyright 2018 Google LLC
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

// Package main is a simple webserver for hosting Open Match Swagger UI.
package main

import (
	"flag"
	"log"
	"os"

	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var (
	port      = flag.Int("port", 51500, "HTTP port number")
	directory = flag.String("directory", "/static", "Directory to serve.")
)

func main() {
	flag.Parse()
	_, err := os.Stat(*directory)
	if err != nil {
		log.Fatalf("Cannot access directory %s, %v", *directory, err)
	}
	serve(*directory, *port)
}

func serve(directory string, port int) {
	mux := &http.ServeMux{}
	mux.Handle("/", http.FileServer(http.Dir(directory)))
	// TODO: Pull these values from the Environment variables or Config.
	bindHandler(mux, "/v1/frontend/", "http://om-frontend:51504")
	bindHandler(mux, "/v1/backend/", "http://om-backend:51505")
	bindHandler(mux, "/v1/mmlogic/", "http://om-mmlogic:51503")
	bindHandler(mux, "/v1/synchronizer/", "http://om-synchronizer:51506")
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	log.Fatal(srv.ListenAndServe())
}

func bindHandler(mux *http.ServeMux, path string, endpoint string) {
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
		log.Printf("URL: %s", req.URL)
	}
	return &httputil.ReverseProxy{Director: director}
}

func mustURLParse(u string) *url.URL {
	url, err := url.Parse(u)
	if err != nil {
		log.Fatalf("failed to parse URL %s, %v", u, err)
	}
	return url
}

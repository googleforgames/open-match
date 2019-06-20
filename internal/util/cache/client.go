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

package cache

import (
	"net/http"
	"sync"

	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
)

type grpcData struct {
	conn *grpc.ClientConn
}

type httpData struct {
	client  *http.Client
	baseURL string
}

// GetHTTPClientFromCache get a httpclient and base url from a sync map and write to the map if the address is first seen.
func GetHTTPClientFromCache(cfg config.View, mmfClients *sync.Map, addr string) (*http.Client, string, error) {
	val, exists := mmfClients.Load(addr)
	data, ok := val.(httpData)
	if !ok || !exists {
		client, baseURL, err := rpc.HTTPClientFromEndpoint(cfg, addr)
		if err != nil {
			return nil, "", err
		}
		data = httpData{client, baseURL}
		mmfClients.Store(addr, data)
	}
	return data.client, data.baseURL, nil
}

// GetGRPCClientFromCache get a grpc client connection from a sync map and write to the mao if the address is first seen.
func GetGRPCClientFromCache(cfg config.View, mmfClients *sync.Map, addr string) (*grpc.ClientConn, error) {
	val, exists := mmfClients.Load(addr)
	data, ok := val.(grpcData)
	if !ok || !exists {
		conn, err := rpc.GRPCClientFromEndpoint(cfg, addr)
		if err != nil {
			return nil, err
		}
		data = grpcData{conn}
		mmfClients.Store(addr, data)
	}

	return data.conn, nil
}

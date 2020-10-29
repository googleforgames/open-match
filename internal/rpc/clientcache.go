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

package rpc

import (
	"net/http"
	"sync"

	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/config"
)

// ClientCache holds GRPC and HTTP clients based on an address.
type ClientCache struct {
	cfg   config.View
	cache *sync.Map
}

type cachedGRPCClient struct {
	client *grpc.ClientConn
}

type cachedHTTPClient struct {
	client  *http.Client
	baseURL string
}

// GetGRPC gets a GRPC client with the address.
func (cc *ClientCache) GetGRPC(address string) (*grpc.ClientConn, error) {
	val, exists := cc.cache.Load(address)
	c, ok := val.(cachedGRPCClient)
	if !ok || !exists {
		conn, err := GRPCClientFromEndpoint(cc.cfg, address)
		if err != nil {
			return nil, err
		}
		c = cachedGRPCClient{
			client: conn,
		}
		cc.cache.Store(address, c)
	}

	return c.client, nil
}

// GetHTTP gets a HTTP client with the address.
func (cc *ClientCache) GetHTTP(address string) (*http.Client, string, error) {
	val, exists := cc.cache.Load(address)
	c, ok := val.(cachedHTTPClient)
	if !ok || !exists {
		client, baseURL, err := HTTPClientFromEndpoint(cc.cfg, address)
		if err != nil {
			return nil, "", err
		}
		c = cachedHTTPClient{client, baseURL}
		cc.cache.Store(address, c)
	}
	return c.client, c.baseURL, nil
}

// NewClientCache creates a cache with all the clients.
func NewClientCache(cfg config.View) *ClientCache {
	return &ClientCache{
		cfg:   cfg,
		cache: &sync.Map{},
	}
}

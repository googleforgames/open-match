/*
Copyright 2019 Google LLC
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package serving

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/GoogleCloudPlatform/open-match/internal/config"
	"github.com/GoogleCloudPlatform/open-match/internal/util/netlistener"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

var (
	serverLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "server",
	})
)

// GrpcHandler binds gRPC services.
type GrpcHandler func(*grpc.Server)

// GrpcProxyHandler binds HTTP handler to gRPC service.
type GrpcProxyHandler func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error

// Params yeah
type Params struct {
	handlersForGrpc      []GrpcHandler
	handlersForGrpcProxy []GrpcProxyHandler
	grpcListener         *netlistener.ListenerHolder
	grpcProxyListener    *netlistener.ListenerHolder
}

// NewParamsFromConfig returns.
func NewParamsFromConfig(cfg config.View, prefix string) (*Params, error) {
	grpcLh, err := netlistener.NewFromPortNumber(cfg.GetInt(prefix + ".port"))
	if err != nil {
		serverLogger.Fatal(err)
		return nil, err
	}
	httpLh, err := netlistener.NewFromPortNumber(cfg.GetInt(prefix + ".proxyport"))
	if err != nil {
		closeErr := grpcLh.Close()
		if closeErr != nil {
			serverLogger.WithFields(logrus.Fields{
				"error": closeErr.Error(),
			}).Info("failed to gRPC close port")
		}
		serverLogger.Fatal(err)
		return nil, err
	}
	p := NewParamsFromListeners(grpcLh, httpLh)

	return p, nil
}

// NewParamsFromListeners returns.
func NewParamsFromListeners(grpcLh *netlistener.ListenerHolder, proxyLh *netlistener.ListenerHolder) *Params {
	return &Params{
		handlersForGrpc:      []GrpcHandler{},
		handlersForGrpcProxy: []GrpcProxyHandler{},
		grpcListener:         grpcLh,
		grpcProxyListener:    proxyLh,
	}
}

// AddHandleFunc binds gRPC service handler and an associated HTTP proxy handler.
func (p *Params) AddHandleFunc(handlerFunc GrpcHandler, grpcProxyHandler GrpcProxyHandler) *Params {
	if handlerFunc != nil {
		p.handlersForGrpc = append(p.handlersForGrpc, handlerFunc)
	}
	if grpcProxyHandler != nil {
		p.handlersForGrpcProxy = append(p.handlersForGrpcProxy, grpcProxyHandler)
	}
	return p
}

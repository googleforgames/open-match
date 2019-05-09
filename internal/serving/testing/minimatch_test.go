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

package testing

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	goTesting "testing"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	pb "open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/serving"
)

func TestNewMiniMatch(t *goTesting.T) {
	ff := &FakeFrontend{}
	mm, closer, err := NewMiniMatch([]*serving.ServerParams{
		{
			BaseLogFields: logrus.Fields{
				"app":       "openmatch",
				"component": "frontend",
			},
			ServicePortConfigName: "api.frontend.port",
			ProxyPortConfigName:   "api.frontend.proxyport",
			Bindings: []serving.BindingFunc{
				func(omServer *serving.OpenMatchServer) {
					omServer.GrpcServer.AddService(func(server *grpc.Server) {
						pb.RegisterFrontendServer(server, ff)
					})
					omServer.GrpcServer.AddProxy(pb.RegisterFrontendHandler)
				},
			},
		},
	})
	if err != nil {
		t.Errorf("could not create Mini Match context %s", err)
	}
	err = mm.Start()
	if err != nil {
		t.Errorf("could not start Mini Match %s", err)
	}

	feClient, err := mm.GetFrontendClient()
	t.Run("FrontendClient Test", func(t *goTesting.T) {
		if err != nil {
			t.Errorf("could not get frontend client %s", err)
		}
		result, err := feClient.CreatePlayer(context.Background(), &pb.CreatePlayerRequest{})
		if err != nil {
			t.Errorf("could not start Mini Match %s", err)
		}
		if result == nil {
			t.Errorf("insert player was not successful %v", result)
		}
	})

	proxyTests := []struct {
		method   string
		endpoint string
		response string
	}{
		// Health check fails when running test cases in parallel
		{
			method:   "GET",
			endpoint: "healthz",
			response: "ok\n",
		},
		{
			method:   "GET",
			endpoint: "nowhere",
			response: "Not Found\n",
		},
	}

	feProxyClient, feBaseURL := mm.GetFrontendProxyClient()
	for _, tt := range proxyTests {
		endpoint := fmt.Sprintf(feBaseURL+"/%s", tt.endpoint)
		t.Run(fmt.Sprintf("ProxyTest-%s-%s", tt.method, endpoint), func(t *goTesting.T) {
			req, err := http.NewRequest(tt.method, endpoint, nil)
			if err != nil {
				t.Errorf("Failed to create new request %v", req)
			}

			resp, err := feProxyClient.Do(req)
			if err != nil {
				t.Errorf("Failed to ping the proxy server %s", err)
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("Failed to read response body %s", err)
			}

			if string(body) != tt.response {
				t.Errorf("Response incorrect, got: %s, expect: %s.", body, tt.response)
			}
		})

	}
	closer()

	// Re-open the port to ensure it's free.
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", mm.Config.GetInt("api.frontend.port")))
	if err != nil {
		t.Errorf("grpc server is still running! Result= %v Error= %s", ln, err)
	}
	result, err := feClient.CreatePlayer(context.Background(), &pb.CreatePlayerRequest{})
	if err == nil {
		t.Errorf("grpc server is still running! Result= %v Error= %s", result, err)
	}
	// Re-open the proxyPort to ensure it's free.
	ln, err = net.Listen("tcp", fmt.Sprintf(":%d", mm.Config.GetInt("api.frontend.proxyport")))
	if err != nil {
		t.Errorf("grpc proxy is still running! Result= %v Error= %s", ln, err)
	}
}

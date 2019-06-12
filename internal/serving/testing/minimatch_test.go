package testing

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	goTesting "testing"

	pb "github.com/googleforgames/open-match/internal/pb"
	"github.com/googleforgames/open-match/internal/serving"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func TestNewMiniMatch(t *goTesting.T) {
	ff := &FakeFrontend{}
	mm, closer, err := NewMiniMatch([]*serving.ServerParams{
		&serving.ServerParams{
			BaseLogFields: log.Fields{
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

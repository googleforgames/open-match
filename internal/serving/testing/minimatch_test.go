package testing

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	goTesting "testing"
	"time"

	pb "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/GoogleCloudPlatform/open-match/internal/serving"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func TestNewMiniMatch(t *goTesting.T) {
	ff := &FakeFrontend{}
	mm, err := NewMiniMatch([]*serving.ServerParams{
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
					omServer.GrpcServer.AddProxy(pb.RegisterFrontendHandlerFromEndpoint)
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
	defer mm.Stop()

	t.Run("FrontendClient Test", func(t *goTesting.T) {
		feClient, err := mm.GetFrontendClient()
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

	testStubs := []struct {
		method   string
		endpoint string
		response string
	}{
		// TODO: Diagnose why the response is 
		//	{code: 1, error: "grpc: the client connection is closing"}
		// {
		// 	method:   "GET",
		//	endpoint: "v1/frontend/players/123",
		//	response: "Not Found",
		//},
		{
			method:   "GET",
			endpoint: "nowhere",
			response: "Not Found\n",
		},
		{
			method:   "GET",
			endpoint: "ping",
			response: "pong",
		},
	}

	for _, stub := range testStubs {
		t.Run(fmt.Sprintf("ProxyTest-%s-%s", stub.method, stub.endpoint), func(t *goTesting.T) {
			apiAddr := fmt.Sprintf("http://:%d/%s", mm.Config.GetInt("api.frontend.proxyport"), stub.endpoint)

			httpClient := &http.Client{
				Timeout: time.Second * 3,
			}
			resp, err := httpClient.Get(apiAddr)
			if err != nil {
				t.Errorf("Failed to ping the proxy server %s", err)
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("Failed to read response body %s", err)
			}

			if string(body) != stub.response {
				t.Errorf("Response incorrect, got: %s, expect: %s.", body, stub.response)
			}
		})

	}

}

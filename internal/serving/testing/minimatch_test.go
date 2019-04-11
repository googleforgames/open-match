package testing

import (
	"context"
	goTesting "testing"

	pb "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/GoogleCloudPlatform/open-match/internal/serving"
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
					omServer.GrpcServer.AddProxy(pb.RegisterFrontendHandlerFromEndpoint)
				},
			},
		},
	})
	defer closer()
	if err != nil {
		t.Errorf("could not create Mini Match context %s", err)
	}
	err = mm.Start()
	if err != nil {
		t.Errorf("could not start Mini Match %s", err)
	}
	defer mm.Stop()
	feClient, err := mm.GetFrontendClient()
	result, err := feClient.CreatePlayer(context.Background(), &pb.CreatePlayerRequest{})
	if err != nil {
		t.Errorf("could not start Mini Match %s", err)
	}
	if result == nil {
		t.Errorf("insert player was not successful %v", result)
	}
}

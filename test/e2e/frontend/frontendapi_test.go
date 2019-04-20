package frontend

import (
	"testing"

	"github.com/GoogleCloudPlatform/open-match/internal/app/frontendapi"

	"github.com/GoogleCloudPlatform/open-match/internal/serving"
	omTesting "github.com/GoogleCloudPlatform/open-match/internal/serving/testing"
)

func TestFrontendStartup(t *testing.T) {
	mm, err := omTesting.NewMiniMatch([]*serving.ServerParams{
		frontendapi.CreateServerParams(),
	})
	if err != nil {
		t.Fatalf("cannot create mini match server, %s", err)
	}
	mm.Start()
	if err != nil {
		t.Fatalf("cannot start mini match server, %s", err)
	}
	defer mm.Stop()

	feClient, err := mm.GetFrontendClient()
	if err != nil {
		t.Errorf("cannot obtain fe client, %s", err)
	}
	if feClient == nil {
		t.Errorf("cannot get fe client, %v", feClient)
	}
}

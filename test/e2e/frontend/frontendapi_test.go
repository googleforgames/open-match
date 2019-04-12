package frontend

import (
	"testing"

	"github.com/GoogleCloudPlatform/open-match/internal/app/frontendapi"

	"github.com/GoogleCloudPlatform/open-match/internal/serving"
	omTesting "github.com/GoogleCloudPlatform/open-match/internal/serving/testing"
)

func TestFrontendStartup(t *testing.T) {
	mm, closer, err := omTesting.NewMiniMatch([]*serving.ServerParams{
		frontendapi.CreateServerParams(),
	})
	defer closer()
	mm.Start()
	defer mm.Stop()

	feClient, err := mm.GetFrontendClient()
	if err != nil {
		t.Errorf("cannot obtain fe client, %s", err)
	}
	if feClient == nil {
		t.Errorf("cannot get fe client, %v", feClient)
	}
}

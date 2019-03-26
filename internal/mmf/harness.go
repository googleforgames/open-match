package mmf

import (
	"log"

	"github.com/GoogleCloudPlatform/open-match/config"
	api "github.com/GoogleCloudPlatform/open-match/internal/pb"
)

func Run(fnArgs *api.Arguments, cfg config.View, mmlogic api.MmLogicClient) error {
	log.Printf("Function called!\n")
	log.Printf("args: %v", &fnArgs)
	return nil
}

package mmf

import (
	"log"

	api "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/spf13/viper"
)

func Run(fnArgs *api.Arguments, cfg *viper.Viper, mmlogic api.MmLogicClient) error {
	log.Printf("Function called!\n")
	log.Printf("args: %v", &fnArgs)
	return nil
}

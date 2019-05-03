package mmlogic

import (
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestGetPool(t *testing.T) {
	cfg := viper.New()
	mredis, err := miniredis.Run()
	assert.NotNil(nil)
}

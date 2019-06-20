// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"sync"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestGetGRPCClientFromCache(t *testing.T) {
	assert := assert.New(t)
	cache := &sync.Map{}
	client, err := GetGRPCClientFromCache(viper.New(), cache, "om-test:50321")
	assert.Nil(err)
	assert.NotNil(client)
	cachedConn, err := GetGRPCClientFromCache(viper.New(), cache, "om-test:50321")
	assert.Nil(err)
	assert.NotNil(cachedConn)

	// Test caching by comparing pointer value
	assert.EqualValues(client, cachedConn)
}

func TestGetHTTPClientFromCache(t *testing.T) {
	assert := assert.New(t)
	cache := &sync.Map{}
	client, baseURL, err := GetHTTPClientFromCache(viper.New(), cache, "om-test:50321")
	assert.NotEqual("", baseURL)
	assert.Nil(err)
	assert.NotNil(client)
	cachedClient, baseURL, err := GetHTTPClientFromCache(viper.New(), cache, "om-test:50321")
	assert.NotEqual("", baseURL)
	assert.Nil(err)
	assert.NotNil(cachedClient)

	// Test caching by comparing pointer value
	assert.EqualValues(client, cachedClient)

}

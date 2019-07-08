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

package rpc

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	fakeHTTPAddress = "http://om-test:54321"
	fakeGRPCAddress = "om-test:54321"
)

func TestGetGRPC(t *testing.T) {
	assert := assert.New(t)

	cc := NewClientCache(viper.New())
	client, err := cc.GetGRPC(fakeGRPCAddress)
	assert.Nil(err)

	cachedClient, err := cc.GetGRPC(fakeGRPCAddress)
	assert.Nil(err)

	// Test caching by comparing pointer value
	assert.EqualValues(client, cachedClient)
}

func TestGetHTTP(t *testing.T) {
	assert := assert.New(t)

	cc := NewClientCache(viper.New())
	client, address, err := cc.GetHTTP(fakeHTTPAddress)
	assert.Nil(err)
	assert.Equal(fakeHTTPAddress, address)

	cachedClient, address, err := cc.GetHTTP(fakeHTTPAddress)
	assert.Nil(err)
	assert.Equal(fakeHTTPAddress, address)

	// Test caching by comparing pointer value
	assert.EqualValues(client, cachedClient)
}

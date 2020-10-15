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
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

const (
	fakeHTTPAddress            = "http://om-test:54321"
	fakeGRPCAddress            = "om-test:54321"
	unavailableFakeGRPCAddress = "om-test-1:54321"
)

func TestGetGRPC(t *testing.T) {
	require := require.New(t)

	cc := NewClientCache(viper.New())
	client, err := cc.GetGRPC(fakeGRPCAddress)
	require.Nil(err)

	cachedClient, err := cc.GetGRPC(fakeGRPCAddress)
	require.Nil(err)

	// Test caching by comparing pointer value
	require.EqualValues(client, cachedClient)
}

func TestGetGRPC_WithUnavailableAddress(t *testing.T) {
	require := require.New(t)

	cc := NewClientCache(viper.New())
	_, err := cc.GetGRPC(unavailableFakeGRPCAddress)
	require.Error(err)

	code := status.Code(err)
	require.Equal(codes.Unavailable, code)
}

func TestGetHTTP(t *testing.T) {
	require := require.New(t)

	cc := NewClientCache(viper.New())
	client, address, err := cc.GetHTTP(fakeHTTPAddress)
	require.Nil(err)
	require.Equal(fakeHTTPAddress, address)

	cachedClient, address, err := cc.GetHTTP(fakeHTTPAddress)
	require.Nil(err)
	require.Equal(fakeHTTPAddress, address)

	// Test caching by comparing pointer value
	require.EqualValues(client, cachedClient)
}

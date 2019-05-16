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

package mmlogic

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
)

func TestQueryTicketsBasic(t *testing.T) {
	assert := assert.New(t)
	vp := viper.New()
	vp.Set("storage.page.size", 1000)

	cfg, closer, err := statestoreTesting.NewRedisForTesting(vp)
	assert.Nil(err)
	defer closer()

	mmlogicService, err := newMmlogic(cfg)
	assert.Nil(err)
	assert.NotNil(mmlogicService)

	// TODO: Add more tests after the client wrapper is done
}
func TestServerBinding(t *testing.T) {
	bs := func(p *rpc.ServerParams) {
		p.AddHandleFunc(func(s *grpc.Server) {
			pb.RegisterMmLogicServer(s, &mmlogicService{})
		}, pb.RegisterMmLogicHandlerFromEndpoint)
	}

	rpcTesting.TestServerBinding(t, bs)
}

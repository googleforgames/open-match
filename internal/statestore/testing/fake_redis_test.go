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

package testing

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"open-match.dev/open-match/internal/statestore"
	utilTesting "open-match.dev/open-match/internal/util/testing"
	"open-match.dev/open-match/pkg/pb"
)

func TestFakeStatestore(t *testing.T) {
	cfg := viper.New()
	closer := New(t, cfg)
	defer closer()
	s := statestore.New(cfg)
	ctx := utilTesting.NewContext(t)

	ticket := &pb.Ticket{
		Id: "abc",
	}
	require.Nil(t, s.CreateTicket(ctx, ticket))
	retrievedTicket, err := s.GetTicket(ctx, "abc")
	require.Nil(t, err)
	require.Equal(t, ticket.Id, retrievedTicket.Id)
}

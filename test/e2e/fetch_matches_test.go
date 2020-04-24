// Copyright 2020 Google LLC
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

package e2e

import (
	"context"
	"io"
	"testing"
	// "time"

	"github.com/stretchr/testify/require"
	// "google.golang.org/grpc"
	"github.com/golang/protobuf/proto"
	"open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
	// "open-match.dev/open-match/test/matchfunction/mmf"
)

func TestHappyPath(t *testing.T) {
	ctx := context.Background()
	om := e2e.New(t)

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)
	t2, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	m := &pb.Match{
		MatchId: "1",
		Tickets: []*pb.Ticket{t1, t2},
	}

	om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		out <- m
		return nil
	})

	om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		p, ok := <-in
		require.True(t, ok)
		require.True(t, proto.Equal(p, m))
		_, ok = <-in
		require.False(t, ok)

		out <- m.MatchId
		return nil
	})

	stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
		Config:  om.MMFConfigGRPC(),
		Profile: &pb.MatchProfile{},
	})
	require.Nil(t, err)

	resp, err := stream.Recv()
	require.Nil(t, err)
	require.True(t, proto.Equal(m, resp.Match))

	resp, err = stream.Recv()
	require.Equal(t, err, io.EOF)
	require.Nil(t, resp)
}

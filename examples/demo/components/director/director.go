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

package director

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"time"

	"google.golang.org/grpc"

	"open-match.dev/open-match/examples/demo/components"
	"open-match.dev/open-match/pkg/pb"
)

func Run(ds *components.DemoShared) {
	for !isContextDone(ds.Ctx) {
		run(ds)
	}
}

func isContextDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

type status struct {
	Status        string
	LatestMatches []*pb.Match
}

func run(ds *components.DemoShared) {
	defer func() {
		r := recover()
		if r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
			}

			ds.Update(status{Status: fmt.Sprintf("Encountered error: %s", err.Error())})
			time.Sleep(time.Second * 10)
		}
	}()

	s := status{}

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Connecting to backend"
	ds.Update(s)

	// See https://open-match.dev/site/docs/guides/api/
	conn, err := grpc.Dial("om-backend.open-match.svc.cluster.local:50505", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	be := pb.NewBackendServiceClient(conn)

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Match Match: Sending Request"
	ds.Update(s)

	var matches []*pb.Match
	{
		req := &pb.FetchMatchesRequest{
			Config: &pb.FunctionConfig{
				Host: "om-function.open-match-demo.svc.cluster.local",
				Port: 50502,
				Type: pb.FunctionConfig_GRPC,
			},
			Profile: &pb.MatchProfile{
				Name: "1v1",
				Pools: []*pb.Pool{
					{
						Name: "Everyone",
					},
				},
			},
		}

		stream, err := be.FetchMatches(ds.Ctx, req)
		if err != nil {
			panic(err)
		}

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				panic(err)
			}
			matches = append(matches, resp.GetMatch())
		}
	}

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Matches Found"
	s.LatestMatches = matches
	ds.Update(s)

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Assigning Players"
	ds.Update(s)

	for _, match := range matches {
		ids := []string{}

		for _, t := range match.Tickets {
			ids = append(ids, t.Id)
		}

		req := &pb.AssignTicketsRequest{
			TicketIds: ids,
			Assignment: &pb.Assignment{
				Connection: fmt.Sprintf("%d.%d.%d.%d:2222", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256)),
			},
		}

		resp, err := be.AssignTickets(ds.Ctx, req)
		if err != nil {
			panic(err)
		}

		_ = resp
	}

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Sleeping"
	ds.Update(s)

	time.Sleep(time.Second * 5)
}

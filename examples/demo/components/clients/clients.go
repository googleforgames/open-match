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

package clients

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"open-match.dev/open-match/examples/demo/components"
	"open-match.dev/open-match/examples/demo/updater"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"
)

func Run(ds *components.DemoShared) {
	u := updater.NewNested(ds.Ctx, ds.Update)
	_ = u

	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("fakeplayer_%d", i)
		go func() {
			for !isContextDone(ds.Ctx) {
				runScenario(ds.Ctx, ds.Cfg, name, u.ForField(name))
			}
		}()
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
	Status string
}

func runScenario(ctx context.Context, cfg config.View, name string, update updater.SetFunc) {
	defer func() {
		r := recover()
		if r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
			}

			update(status{Status: fmt.Sprintf("Encountered error: %s", err.Error())})
			time.Sleep(time.Second * 10)
		}
	}()

	s := status{}

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Main Menu"
	update(s)

	time.Sleep(time.Duration(rand.Int63()) % (time.Second * 5))

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Connecting to Open Match frontend"
	update(s)

	conn, err := rpc.GRPCClientFromConfig(cfg, "api.frontend")
	if err != nil {

	}
	fe := pb.NewFrontendClient(conn)

	//////////////////////////////////////////////////////////////////////////////
	s.Status = "Creating Open Match Ticket"
	update(s)

	var ticketId string

	{
		req := &pb.CreateTicketRequest{
			Ticket: &pb.Ticket{
				Properties: structs.Struct{
					"name":      name,
					"mode.demo": structs.Number(1),
				}.S(),
			},
		}

		resp, err := fe.CreateTicket(ctx, req)
		if err != nil {
			panic(err)
		}
		ticketId = resp.Ticket.Id
	}

	//////////////////////////////////////////////////////////////////////////////
	s.Status = fmt.Sprintf("Waiting match with ticket Id %s", ticketId)
	update(s)

	time.Sleep(time.Second * 10)
}

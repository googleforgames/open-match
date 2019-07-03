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

package e2e

import (
	"testing"

	pb "open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/pkg/structs"
)

func TestServiceHealth(t *testing.T) {
	om, closer := New(t)
	defer closer()
	if err := om.HealthCheck(); err != nil {
		t.Errorf("cannot create ticket, %s", err)
	}
}

func TestGetClients(t *testing.T) {
	om, closer := New(t)
	defer closer()

	if c := om.MustFrontendGRPC(); c == nil {
		t.Error("cannot get frontend client")
	}

	if c := om.MustBackendGRPC(); c == nil {
		t.Error("cannot get backend client")
	}

	if c := om.MustMmLogicGRPC(); c == nil {
		t.Error("cannot get mmlogic client")
	}
}

func TestCreateTicket(t *testing.T) {
	om, closer := New(t)
	defer closer()
	fe := om.MustFrontendGRPC()
	resp, err := fe.CreateTicket(om.Context(), &pb.CreateTicketRequest{
		Ticket: &pb.Ticket{
			Properties: structs.Struct{
				"test-property": structs.Number(1),
			}.S(),
		},
	})
	if err != nil {
		t.Errorf("cannot create ticket, %s", err)
	}
	t.Logf("created ticket %+v", resp)
}

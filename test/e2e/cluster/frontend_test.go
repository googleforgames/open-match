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

package cluster

import (
	"open-match.dev/open-match/test/e2e/cluster/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"open-match.dev/open-match/internal/rpc"
	pb "open-match.dev/open-match/pkg/pb"
	"context"
	structpb "github.com/golang/protobuf/ptypes/struct"
)

func TestCreateTicket(t *testing.T) {
	f := framework.Get()
	t.Logf("Framework %+v", f)
	svc, err := f.KubeClient.CoreV1().Services(f.Namespace).Get("om-frontend", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("cannot get service, %s", err)
	}
	if len(svc.Status.LoadBalancer.Ingress) != 1 {
		t.Fatalf("LoadBalancer for om-frontend does not have exactly 1 config, %v", svc.Status.LoadBalancer.Ingress)
	}
	ipAddress := svc.Status.LoadBalancer.Ingress[0].IP
	port := svc.Spec.Ports[0]
	conn, err := rpc.GRPCClientFromParams(&rpc.ClientParams{
		Hostname: ipAddress,
		Port:     int(port.Port),
	})
	if err != nil {
		t.Fatalf("cannot connect to gRPC %s:%d, %s", ipAddress, port.Port, err)
	}
	fe := pb.NewFrontendClient(conn)
	ctx := context.Background()
	resp, err := fe.CreateTicket(ctx, &pb.CreateTicketRequest{
		Ticket: &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"test-property": {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
			},
		},
	},
	})
	if err != nil {
		t.Errorf("cannot create ticket, %s", err)
	}
	t.Logf("created ticket %+v", resp)
}

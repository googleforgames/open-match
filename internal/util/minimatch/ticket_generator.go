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

// Package minimatch provides utility functions for the minimatch binary.
package minimatch

import (
	"context"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"
)

// TicketGenerator generates 20 tickets with given attributes in range of [0, 10) twice per second
func TicketGenerator(cfg config.View, attributes []string, logger *logrus.Entry) error {
	conn, err := rpc.GRPCClientFromConfig(cfg, "api.frontend")
	if err != nil {
		return err
	}
	client := pb.NewFrontendClient(conn)

	go func() {
		time.Sleep(500 * time.Millisecond)
		for i := 0; i < 20; i++ {
			_, err := client.CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: generateTicket(attributes)})
			if err != nil {
				logger.Error(err)
			}
		}
	}()
	return nil
}

func generateTicket(attributes []string) *pb.Ticket {
	fields := make(map[string]*structpb.Value)
	r := rand.New(rand.NewSource(time.Now().Unix()))

	for i := 0; i < len(attributes); i++ {
		fields[attributes[i]] = &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: r.Float64() * 10}}
	}

	return &pb.Ticket{
		Properties: &structpb.Struct{Fields: fields},
	}
}

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

package scenarios

import (
	"fmt"
	"io"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/util/testing"
	"open-match.dev/open-match/pkg/matchfunction"
	"open-match.dev/open-match/pkg/pb"
)

var (
	queryServiceAddress = "om-query.open-match.svc.cluster.local:50503" // Address of the QueryService Endpoint.

	logger = logrus.WithFields(logrus.Fields{
		"app": "scale",
	})
)

// StatProcessor uses syncMaps to store the stress test metrics and occurrence of errors.
// It can write out the data to an input io.Writer.
type StatProcessor struct {
	em *sync.Map
	sm *sync.Map
}

// NewStatProcessor returns an initialized StatProcessor
func NewStatProcessor() *StatProcessor {
	return &StatProcessor{
		em: &sync.Map{},
		sm: &sync.Map{},
	}
}

// SetStat sets the value for a key
func (e StatProcessor) SetStat(k string, v interface{}) {
	e.sm.Store(k, v)
}

// IncrementStat atomically increments the value of a key by delta
func (e StatProcessor) IncrementStat(k string, delta interface{}) {
	statRead, ok := e.sm.Load(k)
	if !ok {
		statRead = 0
	}

	switch delta.(type) {
	case int:
		e.sm.Store(k, statRead.(int)+delta.(int))
	case float32:
		e.sm.Store(k, statRead.(float32)+delta.(float32))
	case float64:
		e.sm.Store(k, statRead.(float64)+delta.(float64))
	default:
		logger.Errorf("IncrementStat: type %T not supported", delta)
	}
}

// RecordError atomically records the occurrence of input errors
func (e StatProcessor) RecordError(desc string, err error) {
	errMsg := fmt.Sprintf("%s: %s", desc, err.Error())
	errRead, ok := e.em.Load(errMsg)
	if !ok {
		errRead = 0
	}
	e.em.Store(errMsg, errRead.(int)+1)
}

// Log writes the formatted errors and metrics to the input writer
func (e StatProcessor) Log(w io.Writer) {
	e.sm.Range(func(k interface{}, v interface{}) bool {
		w.Write([]byte(fmt.Sprintf("%s: %d \n", k, v)))
		return true
	})
	e.em.Range(func(k interface{}, v interface{}) bool {
		w.Write([]byte(fmt.Sprintf("%s: %d \n", k, v)))
		return true
	})
}

func getQueryServiceGRPCClient() pb.QueryServiceClient {
	conn, err := grpc.Dial(queryServiceAddress, testing.NewGRPCDialOptions(logger)...)
	if err != nil {
		logger.Fatalf("Failed to connect to Open Match, got %v", err)
	}
	return pb.NewQueryServiceClient(conn)
}

func queryPoolsWrapper(mmf func(req *pb.MatchProfile, pools map[string][]*pb.Ticket) ([]*pb.Match, error)) matchFunction {
	var q pb.QueryServiceClient
	var startQ sync.Once

	return func(req *pb.RunRequest, stream pb.MatchFunction_RunServer) error {
		startQ.Do(func() {
			q = getQueryServiceGRPCClient()
		})

		poolTickets, err := matchfunction.QueryPools(stream.Context(), q, req.GetProfile().GetPools())
		if err != nil {
			return err
		}

		proposals, err := mmf(req.GetProfile(), poolTickets)
		if err != nil {
			return err
		}

		logger.WithFields(logrus.Fields{
			"proposals": proposals,
		}).Trace("proposals returned by match function")

		for _, proposal := range proposals {
			if err := stream.Send(&pb.RunResponse{Proposal: proposal}); err != nil {
				return err
			}
		}

		return nil
	}
}

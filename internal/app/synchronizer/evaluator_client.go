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

package synchronizer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/golang/protobuf/jsonpb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"
)

type evaluator interface {
	evaluate(context.Context, []*pb.Match) ([]*pb.Match, error)
}

type clientType int

// Different type of clients supported for evaluation.
const (
	none clientType = iota
	grpcEvaluator
	httpEvaluator
)

var (
	evaluatorClientLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.synchronizer.evaluator_client",
	})
)

type evaluatorClient struct {
	clType     clientType
	cfg        config.View
	grpcClient pb.EvaluatorClient
	httpClient *http.Client
	baseURL    string
	m          sync.Mutex
}

// evaluate method triggers execution of user evaluation function. It initializes the configured
// GRPC or HTTP client. If the client was already initialized, initialize is a no-op.
func (ec *evaluatorClient) evaluate(ctx context.Context, proposals []*pb.Match) (result []*pb.Match, err error) {
	if err := ec.initialize(); err != nil {
		return nil, err
	}

	// Call the appropriate client's evaluation function.
	switch ec.clType {
	case grpcEvaluator:
		return ec.grpcEvaluate(ctx, proposals)
	case httpEvaluator:
		return ec.httpEvaluate(ctx, proposals)
	default:
		return nil, status.Error(codes.Unavailable, "failed to initialize a connection to the evaluator")
	}
}

// initialize sets up the configured GRPC or HTTP client the first time it is called. Consequent
// invocations after a successful client setup are a no-op. It attempts to initialize either GRPC
// or HTTP clients. If both are defined, it prefers GRPC client.
func (ec *evaluatorClient) initialize() error {
	ec.m.Lock()
	defer ec.m.Unlock()
	if ec.clType != none {
		// Client already initialized.
		return nil
	}

	// Evaluator client not initialized. Attempt to initialize GRPC client or HTTP client.
	grpcAddr := fmt.Sprintf("%s:%d", ec.cfg.GetString("api.evaluator.hostname"), ec.cfg.GetInt64("api.evaluator.grpcport"))
	conn, err := rpc.GRPCClientFromEndpoint(ec.cfg, grpcAddr)
	if err == nil {
		evaluatorClientLogger.WithFields(logrus.Fields{
			"endpoint": grpcAddr,
		}).Info("Created a GRPC client for input endpoint.")
		ec.grpcClient = pb.NewEvaluatorClient(conn)
		ec.clType = grpcEvaluator
	}

	evaluatorClientLogger.WithError(err).Errorf("failed to get a GRPC client from the endpoint %v", grpcAddr)
	httpAddr := fmt.Sprintf("%s:%d", ec.cfg.GetString("api.evaluator.hostname"), ec.cfg.GetInt64("api.evaluator.httpport"))
	fmt.Println(httpAddr)
	client, baseURL, err := rpc.HTTPClientFromEndpoint(ec.cfg, httpAddr)
	if err != nil {
		evaluatorClientLogger.WithError(err).Errorf("failed to get a HTTP client from the endpoint %v", httpAddr)
		return err
	}

	evaluatorClientLogger.WithFields(logrus.Fields{
		"endpoint": httpAddr,
	}).Info("Created a HTTP client for input endpoint.")
	ec.httpClient = client
	ec.baseURL = baseURL
	return nil
}

func (ec *evaluatorClient) grpcEvaluate(ctx context.Context, proposals []*pb.Match) ([]*pb.Match, error) {
	stream, err := ec.grpcClient.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	for _, proposal := range proposals {
		if err = stream.Send(&pb.EvaluateRequest{Match: proposal}); err != nil {
			return nil, err
		}
	}

	if err = stream.CloseSend(); err != nil {
		logger.Errorf("failed to close the send stream: %s", err.Error())
	}

	var results = []*pb.Match{}
	for {
		// TODO: add grpc timeouts for this call.
		resp, err := stream.Recv()
		if err == io.EOF {
			// read done.
			break
		}
		if err != nil {
			return nil, err
		}
		results = append(results, resp.GetMatch())
	}

	return results, nil
}

func (ec *evaluatorClient) httpEvaluate(ctx context.Context, proposals []*pb.Match) ([]*pb.Match, error) {
	reqr, reqw := io.Pipe()
	proposalIDs := getMatchIds(proposals)
	waitc := make(chan error)

	go func() {
		var m jsonpb.Marshaler
		defer close(waitc)
		defer func() {
			sendErr := reqw.Close()
			if sendErr != nil {
				logger.Warning("failed to close response body read closer")
			}
		}()
		for _, proposal := range proposals {
			buf, sendErr := m.MarshalToString(&pb.EvaluateRequest{Match: proposal})
			if sendErr != nil {
				waitc <- status.Errorf(codes.FailedPrecondition, "failed to marshal proposal to string: %s", sendErr.Error())
				return
			}
			_, sendErr = io.WriteString(reqw, buf)
			if sendErr != nil {
				waitc <- status.Errorf(codes.FailedPrecondition, "failed to write proto string to io writer: %s", sendErr.Error())
				return
			}
		}
	}()

	req, err := http.NewRequest("POST", ec.baseURL+"/v1/evaluator/matches:evaluate", reqr)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed to create evaluator http request for proposals %s: %s", proposalIDs, err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Transfer-Encoding", "chunked")

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get response from evaluator for proposals %s: %s", proposalIDs, err.Error())
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			logger.WithError(err).Warning("failed to close response body read closer")
		}
	}()

	var got = []*pb.Match{}
	dec := json.NewDecoder(resp.Body)
	for {
		var item struct {
			Result json.RawMessage        `json:"result"`
			Error  map[string]interface{} `json:"error"`
		}
		err = dec.Decode(&item)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, status.Errorf(codes.Unavailable, "failed to read response from HTTP JSON stream: %s", err.Error())
		}
		if len(item.Error) != 0 {
			return nil, status.Errorf(codes.Unavailable, "failed to execute evaluator.Evaluate: %v", item.Error)
		}
		resp := &pb.EvaluateResponse{}
		if err := jsonpb.UnmarshalString(string(item.Result), resp); err != nil {
			return nil, status.Errorf(codes.Unavailable, "failed to execute jsonpb.UnmarshalString(%s, &proposal): %v.", item.Result, err)
		}
		got = append(got, resp.GetMatch())
	}

	if err, ok := <-waitc; ok && err != nil {
		return nil, err
	}
	return got, nil
}

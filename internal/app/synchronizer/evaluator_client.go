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
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"
)

var (
	evaluatorClientLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.synchronizer.evaluator_client",
	})
)

type evaluator interface {
	evaluate(context.Context, <-chan []*pb.Match, chan<- string) error
}

var errNoEvaluatorType = status.Errorf(codes.FailedPrecondition, "unable to determine evaluator type, either api.evaluator.grpcport or api.evaluator.httpport must be specified in the config")

func newEvaluator(cfg config.View) evaluator {
	newInstance := func(cfg config.View) (interface{}, func(), error) {
		// grpc is preferred over http.
		if cfg.IsSet("api.evaluator.grpcport") {
			return newGrpcEvaluator(cfg)
		}
		if cfg.IsSet("api.evaluator.httpport") {
			return newHTTPEvaluator(cfg)
		}
		return nil, nil, errNoEvaluatorType
	}

	return &deferredEvaluator{
		cacher: config.NewCacher(cfg, newInstance),
	}
}

type deferredEvaluator struct {
	cacher *config.Cacher
}

func (de *deferredEvaluator) evaluate(ctx context.Context, pc <-chan []*pb.Match, acceptedIds chan<- string) error {
	e, err := de.cacher.Get()
	if err != nil {
		return err
	}

	err = e.(evaluator).evaluate(ctx, pc, acceptedIds)
	if err != nil {
		de.cacher.ForceReset()
	}
	return err
}

type grcpEvaluatorClient struct {
	evaluator pb.EvaluatorClient
}

func newGrpcEvaluator(cfg config.View) (evaluator, func(), error) {
	grpcAddr := fmt.Sprintf("%s:%d", cfg.GetString("api.evaluator.hostname"), cfg.GetInt64("api.evaluator.grpcport"))
	conn, err := rpc.GRPCClientFromEndpoint(cfg, grpcAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create grpc evaluator client: %w", err)
	}

	evaluatorClientLogger.WithFields(logrus.Fields{
		"endpoint": grpcAddr,
	}).Info("Created a GRPC client for evaluator endpoint.")

	close := func() {
		err := conn.Close()
		if err != nil {
			logger.WithError(err).Warning("Error closing synchronizer client.")
		}
	}

	return &grcpEvaluatorClient{
		evaluator: pb.NewEvaluatorClient(conn),
	}, close, nil
}

func (ec *grcpEvaluatorClient) evaluate(ctx context.Context, pc <-chan []*pb.Match, acceptedIds chan<- string) error {
	eg, ctx := errgroup.WithContext(ctx)

	var stream pb.Evaluator_EvaluateClient
	{ // prevent shadowing err later
		var err error
		stream, err = ec.evaluator.Evaluate(ctx)
		if err != nil {
			return fmt.Errorf("error starting evaluator call: %w", err)
		}
	}

	matchIDs := &sync.Map{}
	eg.Go(func() error {
		for proposals := range pc {
			for _, proposal := range proposals {

				if _, ok := matchIDs.LoadOrStore(proposal.GetMatchId(), true); ok {
					return fmt.Errorf("multiple match functions used same match_id: \"%s\"", proposal.GetMatchId())
				}
				if err := stream.Send(&pb.EvaluateRequest{Match: proposal}); err != nil {
					return fmt.Errorf("failed to send request to evaluator, desc: %w", err)
				}
			}
		}

		if err := stream.CloseSend(); err != nil {
			return fmt.Errorf("failed to close the send direction of evaluator stream, desc: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		for {
			// TODO: add grpc timeouts for this call.
			resp, err := stream.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("failed to get response from evaluator client, desc: %w", err)
			}

			v, ok := matchIDs.Load(resp.GetMatchId())
			if !ok {
				return fmt.Errorf("evaluator returned match_id \"%s\" which does not correspond to its any match in its input", resp.GetMatchId())
			}
			if !v.(bool) {
				return fmt.Errorf("evaluator returned same match_id twice: \"%s\"", resp.GetMatchId())
			}
			matchIDs.Store(resp.GetMatchId(), false)
			acceptedIds <- resp.GetMatchId()
		}
	})

	err := eg.Wait()
	if err != nil {
		return err
	}
	return nil
}

type httpEvaluatorClient struct {
	httpClient *http.Client
	baseURL    string
}

func newHTTPEvaluator(cfg config.View) (evaluator, func(), error) {
	httpAddr := fmt.Sprintf("%s:%d", cfg.GetString("api.evaluator.hostname"), cfg.GetInt64("api.evaluator.httpport"))
	client, baseURL, err := rpc.HTTPClientFromEndpoint(cfg, httpAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get a HTTP client from the endpoint %v: %w", httpAddr, err)
	}

	evaluatorClientLogger.WithFields(logrus.Fields{
		"endpoint": httpAddr,
	}).Info("Created a HTTP client for evaluator endpoint.")

	close := func() {
		client.CloseIdleConnections()
	}

	return &httpEvaluatorClient{
		httpClient: client,
		baseURL:    baseURL,
	}, close, nil
}

func (ec *httpEvaluatorClient) evaluate(ctx context.Context, pc <-chan []*pb.Match, acceptedIds chan<- string) error {
	reqr, reqw := io.Pipe()
	var wg sync.WaitGroup
	wg.Add(1)

	sc := make(chan error, 1)
	defer close(sc)
	go func() {
		var m jsonpb.Marshaler
		defer func() {
			wg.Done()
			if reqw.Close() != nil {
				logger.Warning("failed to close response body read closer")
			}
		}()
		for proposals := range pc {
			for _, proposal := range proposals {
				buf, err := m.MarshalToString(&pb.EvaluateRequest{Match: proposal})
				if err != nil {
					sc <- status.Errorf(codes.FailedPrecondition, "failed to marshal proposal to string: %s", err.Error())
					return
				}
				_, err = io.WriteString(reqw, buf)
				if err != nil {
					sc <- status.Errorf(codes.FailedPrecondition, "failed to write proto string to io writer: %s", err.Error())
					return
				}
			}
		}
	}()

	req, err := http.NewRequest("POST", ec.baseURL+"/v1/evaluator/matches:evaluate", reqr)
	if err != nil {
		return status.Errorf(codes.Aborted, "failed to create evaluator http request, desc: %s", err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Transfer-Encoding", "chunked")

	resp, err := ec.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return status.Errorf(codes.Aborted, "failed to get response from evaluator, desc: %s", err.Error())
	}
	defer func() {
		if resp.Body.Close() != nil {
			logger.Warning("failed to close response body read closer")
		}
	}()

	wg.Add(1)
	rc := make(chan error, 1)
	defer close(rc)
	go func() {
		defer wg.Done()

		dec := json.NewDecoder(resp.Body)
		for {
			var item struct {
				Result json.RawMessage        `json:"result"`
				Error  map[string]interface{} `json:"error"`
			}
			err := dec.Decode(&item)
			if err == io.EOF {
				break
			}
			if err != nil {
				rc <- status.Errorf(codes.Unavailable, "failed to read response from HTTP JSON stream: %s", err.Error())
				return
			}
			if len(item.Error) != 0 {
				rc <- status.Errorf(codes.Unavailable, "failed to execute evaluator.Evaluate: %v", item.Error)
				return
			}
			resp := &pb.EvaluateResponse{}
			if err = jsonpb.UnmarshalString(string(item.Result), resp); err != nil {
				rc <- status.Errorf(codes.Unavailable, "failed to execute jsonpb.UnmarshalString(%s, &proposal): %v.", item.Result, err)
				return
			}
			acceptedIds <- resp.GetMatchId()
		}
	}()

	wg.Wait()
	if len(sc) != 0 {
		return <-sc
	}
	if len(rc) != 0 {
		return <-rc
	}
	return nil
}

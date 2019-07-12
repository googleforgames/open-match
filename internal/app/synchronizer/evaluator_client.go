package synchronizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"
)

type evaluator interface {
	evaluate([]*pb.Match) ([]*pb.Match, error)
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
func (ec *evaluatorClient) evaluate(proposals []*pb.Match) (result []*pb.Match, err error) {
	if err := ec.initialize(); err != nil {
		return nil, err
	}

	// Call the appropriate client's evaluation function.
	switch ec.clType {
	case grpcEvaluator:
		return ec.grpcEvaluate(proposals)
	case httpEvaluator:
		return ec.httpEvaluate(proposals)
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
		return nil
	}

	evaluatorClientLogger.WithError(err).Errorf("failed to get a GRPC client from the endpoint %v", grpcAddr)
	httpAddr := fmt.Sprintf("%s:%d", ec.cfg.GetString("api.evaluator.hostname"), ec.cfg.GetInt64("api.evaluator.httpport"))
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

func (ec *evaluatorClient) grpcEvaluate(proposals []*pb.Match) (results []*pb.Match, err error) {
	resp, err := ec.grpcClient.Evaluate(context.Background(), &pb.EvaluateRequest{Matches: proposals})
	if err != nil {
		return nil, err
	}

	return resp.GetMatches(), nil
}

func (ec *evaluatorClient) httpEvaluate(proposals []*pb.Match) (results []*pb.Match, err error) {
	proposalIDs := getMatchIds(proposals)
	jsonProposals, err := json.Marshal(proposals)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed to marshal proposals to string for proposals %s: %s", proposalIDs, err.Error())
	}

	reqBody, err := json.Marshal(map[string]json.RawMessage{"match": jsonProposals})
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed to marshal request body for proposals %s: %s", proposalIDs, err.Error())
	}

	req, err := http.NewRequest("POST", ec.baseURL+"/v1/matches:evaluate", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed to create evaluator http request for proposals %s: %s", proposalIDs, err.Error())
	}

	resp, err := ec.httpClient.Do(req.WithContext(context.Background()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get response from evaluator for proposals %s: %s", proposalIDs, err.Error())
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			evaluatorClientLogger.WithError(err).Warning("failed to close response body read closer")
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition,
			"failed to read from response body for proposals %s: %s", proposalIDs, err.Error())
	}

	pbResp := &pb.EvaluateResponse{}
	err = json.Unmarshal(body, pbResp)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition,
			"failed to unmarshal response body to response pb for proposals %s: %s", proposalIDs, err.Error())
	}

	return pbResp.GetMatches(), nil
}

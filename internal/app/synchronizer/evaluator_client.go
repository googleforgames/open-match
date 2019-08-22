package synchronizer

import (
	"context"
	"fmt"
	"io"
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

func (ec *evaluatorClient) grpcEvaluate(ctx context.Context, proposals []*pb.Match) ([]*pb.Match, error) {
	stream, err := ec.grpcClient.Evaluate(ctx)

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

func (ec *evaluatorClient) httpEvaluate(ctx context.Context, proposals []*pb.Match) (results []*pb.Match, err error) {
	// TODO: Implement a bidirectional HTTP streaming client in v0.7
	return nil, nil
}

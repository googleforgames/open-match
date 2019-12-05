package backend

import (
	"context"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/ipb"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/internal/util"
	"open-match.dev/open-match/pkg/pb"
)

var (
	mMatchEvaluations = telemetry.Counter("backend/matches_evaluated", "matches evaluated")
)

type synchronizerClient struct {
	enabled func() bool
	cacher  *config.Cacher
}

func newSynchronizerClient(cfg config.View) *synchronizerClient {
	newInstance := func(cfg config.View) (interface{}, error) {
		conn, err := rpc.GRPCClientFromConfig(cfg, "api.synchronizer")
		if err != nil {
			return nil, err
		}

		return ipb.NewSynchronizerClient(conn), nil
	}

	enabled := func() bool {
		return cfg.GetBool("synchronizer.enabled")
	}

	return &synchronizerClient{
		enabled: enabled,
		cacher:  config.NewCacher(cfg, newInstance),
	}
}

// register calls the Register method on Synchronizer service. It only triggers this
// if the synchronizer is enabled. The first attempt to call the synchronizer service
// establishes a connection. Consequent requests use the cached connection.
func (sc *synchronizerClient) register(ctx context.Context) (string, error) {
	if !sc.enabled() {
		// Synchronizer is disabled. Succeed the call without returning any ID.
		return "", nil
	}

	client, err := sc.cacher.Get()
	if err != nil {
		return "", err
	}

	resp, err := client.(ipb.SynchronizerClient).Register(ctx, &ipb.RegisterRequest{})
	if err != nil {
		return "", err
	}

	return resp.GetId(), nil
}

// evaluate calls the EvaluateProposals method on Synchronizer service. It only triggers
// this if the synchronizer is enabled. The first attempt to call the synchronizer service
// establishes a connection. Consequent requests use the cached connection.
func (sc *synchronizerClient) evaluate(ctx context.Context, id string, proposals []*pb.Match) ([]*pb.Match, error) {
	if !sc.enabled() {
		// Synchronizer is disabled. Return all the proposals as results. This is only temporary.
		// After the synchronizer is implememnted, it will be mandatory and this check will be removed.
		return proposals, nil
	}

	client, err := sc.cacher.Get()
	if err != nil {
		return nil, err
	}

	ctx, err = util.AppendSynchronizerContextID(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	stream, err := client.(ipb.SynchronizerClient).EvaluateProposals(ctx)
	for _, proposal := range proposals {
		if err = stream.Send(&ipb.EvaluateProposalsRequest{Id: id, Match: proposal}); err != nil {
			return nil, err
		}
	}

	if err = stream.CloseSend(); err != nil {
		logger.Errorf("failed to close the send stream: %s", err.Error())
	}

	var results = []*pb.Match{}
	for {
		resp, recvErr := stream.Recv()
		if recvErr == io.EOF {
			// read done
			break
		}
		if recvErr != nil {
			return nil, recvErr
		}
		results = append(results, resp.GetMatch())
	}

	telemetry.RecordNUnitMeasurement(ctx, mMatchEvaluations, int64(len(results)))
	return results, nil
}

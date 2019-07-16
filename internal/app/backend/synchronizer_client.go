package backend

import (
	"context"
	"sync"

	"open-match.dev/open-match/internal/config"
	ipb "open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

var (
	mMatchEvaluations = telemetry.Counter("backend/matches_evaluated", "matches evaluated")
)

type synchronizerClient struct {
	cfg          config.View
	synchronizer ipb.SynchronizerClient
	m            sync.Mutex
}

// register calls the Register method on Synchronizer service. It only triggers this
// if the synchronizer is enabled. The first attempt to call the synchronizer service
// establishes a connection. Consequent requests use the cached connection.
func (sc *synchronizerClient) register(ctx context.Context) (string, error) {
	if !sc.cfg.GetBool("synchronizer.enabled") {
		// Synchronizer is disabled. Succeed the call without returning any ID.
		return "", nil
	}

	if err := sc.initialize(); err != nil {
		// Failed to connect to the synchronizer service.
		return "", err
	}

	resp, err := sc.synchronizer.Register(ctx, &ipb.RegisterRequest{})
	if err != nil {
		return "", err
	}

	return resp.GetId(), nil
}

// evaluate calls the EvaluateProposals method on Synchronizer service. It only triggers
// this if the synchronizer is enabled. The first attempt to call the synchronizer service
// establishes a connection. Consequent requests use the cached connection.
func (sc *synchronizerClient) evaluate(ctx context.Context, id string, proposals []*pb.Match) ([]*pb.Match, error) {
	if !sc.cfg.GetBool("synchronizer.enabled") {
		// Synchronizer is disabled. Return all the proposals as results. This is only temporary.
		// After the synchronizer is implememnted, it will be mandatory and this check will be removed.
		return proposals, nil
	}

	if err := sc.initialize(); err != nil {
		// Failed to connect to the synchronizer service.
		return nil, err
	}

	resp, err := sc.synchronizer.EvaluateProposals(ctx, &ipb.EvaluateProposalsRequest{
		Id:      id,
		Matches: proposals})
	if err != nil {
		return nil, err
	}

	telemetry.IncrementCounterN(ctx, mMatchEvaluations, len(resp.Matches))
	return resp.Matches, nil
}

// initialize attempts to connect to the Sychronizer service. If the connection is
// successful, the client is cached and reused for all future communication.
func (sc *synchronizerClient) initialize() error {
	sc.m.Lock()
	defer sc.m.Unlock()
	if sc.synchronizer == nil {
		conn, err := rpc.GRPCClientFromConfig(sc.cfg, "api.synchronizer")
		if err != nil {
			return err
		}

		sc.synchronizer = ipb.NewSynchronizerClient(conn)
	}

	return nil
}

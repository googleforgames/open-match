package backend

import (
	"context"

	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/ipb"
	"open-match.dev/open-match/internal/rpc"
)

type synchronizerClient struct {
	cacher *config.Cacher
}

func newSynchronizerClient(cfg config.View) *synchronizerClient {
	newInstance := func(cfg config.View) (interface{}, error) {
		conn, err := rpc.GRPCClientFromConfig(cfg, "api.synchronizer")
		if err != nil {
			return nil, err
		}

		return ipb.NewSynchronizerClient(conn), nil
	}

	return &synchronizerClient{
		cacher: config.NewCacher(cfg, newInstance),
	}
}

type synchronizerStream interface {
	Send(*ipb.SynchronizeRequest) error
	Recv() (*ipb.SynchronizeResponse, error)
	CloseSend() error
}

func (sc *synchronizerClient) synchronize(ctx context.Context) (synchronizerStream, error) {
	client, err := sc.cacher.Get()
	if err != nil {
		return nil, err
	}
	return client.(ipb.SynchronizerClient).Synchronize(ctx)
}

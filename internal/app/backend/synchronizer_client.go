package backend

import (
	"context"

	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/ipb"
	"open-match.dev/open-match/internal/rpc"
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

type synchronizerStream interface {
	Send(*ipb.SynchronizeRequest) error
	Recv() (*ipb.SynchronizeResponse, error)
	CloseSend() error
}

func (sc *synchronizerClient) synchronize(ctx context.Context) (synchronizerStream, error) {
	if sc.enabled() {
		client, err := sc.cacher.Get()
		if err != nil {
			return nil, err
		}
		return client.(ipb.SynchronizerClient).Synchronize(ctx)
	}
	panic("Synchronizer being disabled is not supported at this time.")
}

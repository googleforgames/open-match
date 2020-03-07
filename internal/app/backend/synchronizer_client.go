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
	newInstance := func(cfg config.View) (interface{}, func(), error) {
		conn, err := rpc.GRPCClientFromConfig(cfg, "api.synchronizer")
		if err != nil {
			return nil, nil, err
		}

		close := func() {
			err := conn.Close()
			if err != nil {
				logger.WithError(err).Warning("Error closing synchronizer client.")
			}
		}

		return ipb.NewSynchronizerClient(conn), close, nil
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

package serving

import (
	"github.com/GoogleCloudPlatform/open-match/internal/config"
	"github.com/GoogleCloudPlatform/open-match/internal/signal"
	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
)

// OpenMatchServer contains the context of a standard Open Match Server.
type OpenMatchServer struct {
	GrpcServer *GrpcWrapper
	Logger     *log.Entry
	RedisPool  *redis.Pool
	Config     config.View
}

// CopyFrom copies the state of an OpenMatchServer from the source server.
func (oms *OpenMatchServer) CopyFrom(src *OpenMatchServer) {
	oms.GrpcServer = src.GrpcServer
	oms.Logger = src.Logger
	oms.RedisPool = src.RedisPool
	oms.Config = src.Config
}

// Start the Open Match Server
func (oms *OpenMatchServer) Start() error {
	return oms.GrpcServer.Start()
}

// Stop the Open Match Server
func (oms *OpenMatchServer) Stop() error {
	redisErr := oms.RedisPool.Close()
	grpcStopErr := oms.GrpcServer.Stop()
	if redisErr != nil {
		return redisErr
	}
	return grpcStopErr
}

// MustServeForever is a convenience method for a production server to start serving and wait for termination signal.
func MustServeForever(params *ServerParams) {
	mustServeOpenMatchForever(New(params))
}

// MustServeForeverMulti is a convenience method for a production server to start serving multiple handlers and wait for termination signal.
func MustServeForeverMulti(params []*ServerParams) {
	mustServeOpenMatchForever(NewMulti(params))
}

func mustServeOpenMatchForever(omServer *OpenMatchServer, err error) {
	if err != nil {
		// New and NewMulti should fatal out before this point. Log via stdout since we don't have a guaranteed log context.
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Info("Cannot construct the gRPC server.")
		return
	}
	logger := omServer.Logger
	defer func() {
		err := omServer.Stop()
		if err != nil {
			logger.WithFields(log.Fields{"error": err.Error()}).Infof("Server shutdown error, %s.", err)
		}
	}()
	// Start serving traffic.
	err = omServer.Start()
	if err != nil {
		logger.WithFields(log.Fields{"error": err.Error()}).Fatal("Failed to start server")
	}

	// Exit when we see a signal
	wait, _ := signal.New()
	wait()
	logger.Info("Shutting down server")
}

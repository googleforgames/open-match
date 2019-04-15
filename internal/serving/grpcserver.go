package serving

import (
	"net"

	"github.com/GoogleCloudPlatform/open-match/internal/port"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"

	"google.golang.org/grpc"
)

// GrpcWrapper is a decoration around the standard GRPC server that sets up a bunch of things common to Open Match servers.
type GrpcWrapper struct {
	port         *port.Port
	handlerFuncs []func(*grpc.Server)
	server       *grpc.Server
	ln           net.Listener
	logger       *log.Entry
	grpcAwaiter  chan error
}

// NewGrpcServer creates a new GrpcWrapper.
func NewGrpcServer(port *port.Port, logger *log.Entry) *GrpcWrapper {
	return &GrpcWrapper{
		port:         port,
		logger:       logger,
		handlerFuncs: []func(*grpc.Server){},
	}
}

// AddService adds a service registration function to be run when the server is created.
func (gw *GrpcWrapper) AddService(handlerFunc func(*grpc.Server)) {
	gw.handlerFuncs = append(gw.handlerFuncs, handlerFunc)
}

// Start begins the gRPC server.
func (gw *GrpcWrapper) Start() error {
	if gw.ln != nil {
		return nil
	}
	ln, err := gw.port.Obtain()
	if err != nil {
		gw.logger.WithFields(log.Fields{
			"error": err.Error(),
			"port":  gw.port.Number(),
		}).Error("net.Listen() error")
		return err
	}
	gw.ln = ln

	gw.logger.WithFields(log.Fields{"port": gw.port.Number()}).Info("TCP net listener initialized")

	server := grpc.NewServer(grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	for _, handlerFunc := range gw.handlerFuncs {
		handlerFunc(server)
	}
	gw.server = server
	gw.grpcAwaiter = make(chan error)

	go func() {
		gw.logger.Infof("Serving gRPC on :%d", gw.port.Number())
		err := gw.server.Serve(ln)
		gw.grpcAwaiter <- err
		if err != nil {
			gw.logger.WithFields(log.Fields{"error": err.Error()}).Error("gRPC serve() error")
		}
	}()

	return nil
}

// WaitForTermination blocks until the gRPC server is shutdown.
func (gw *GrpcWrapper) WaitForTermination() error {
	var err error
	if gw.grpcAwaiter != nil {
		err = <-gw.grpcAwaiter
		close(gw.grpcAwaiter)
		gw.grpcAwaiter = nil
	}
	return err
}

// Stop gracefully shutsdown the gRPC server.
func (gw *GrpcWrapper) Stop() error {
	if gw.server == nil {
		return nil
	}
	gw.server.GracefulStop()
	portErr := gw.ln.Close()
	grpcErr := gw.WaitForTermination()
	gw.server = nil
	gw.ln = nil
	if grpcErr != nil {
		return grpcErr
	}
	return portErr
}

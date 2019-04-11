package serving

import (
	"net"
	"net/http"
	"strconv"

	"github.com/GoogleCloudPlatform/open-match/internal/util/netlistener"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// GrpcWrapper is a decoration around the standard GRPC server that sets up a bunch of things common to Open Match servers.
type GrpcWrapper struct {
	serviceLh           *netlistener.ListenerHolder
	proxyLh             *netlistener.ListenerHolder
	servicehandlerFuncs []func(*grpc.Server)
	proxyhandlerFunc    func(proxyEndpoint string) (*runtime.ServeMux, error)
	server              *grpc.Server
	proxy               *http.Server
	ln                  net.Listener
	logger              *log.Entry
	grpcAwaiter         chan error
}

// NewGrpcServer creates a new GrpcWrapper.
func NewGrpcServer(serviceLh, proxyLh *netlistener.ListenerHolder, logger *log.Entry) *GrpcWrapper {
	return &GrpcWrapper{
		serviceLh:           serviceLh,
		proxyLh:             proxyLh,
		logger:              logger,
		servicehandlerFuncs: []func(*grpc.Server){},
	}
}

// AddService adds a service registration function to be run when the server is created.
func (gw *GrpcWrapper) AddService(handlerFunc func(*grpc.Server)) {
	gw.servicehandlerFuncs = append(gw.servicehandlerFuncs, handlerFunc)
}

func (gw *GrpcWrapper) AddProxy(proxyhandlerFunc func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error) {
	gw.proxyhandlerFunc = func(endpoint string) (*runtime.ServeMux, error) {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		mux := runtime.NewServeMux()
		return mux, proxyhandlerFunc(ctx, mux, endpoint, []grpc.DialOption{grpc.WithInsecure()})
	}
}

// Start begins the gRPC server.
func (gw *GrpcWrapper) Start() error {
	// Starting gRPC server
	if gw.ln != nil {
		return nil
	}
	
	ln, err := gw.serviceLh.Obtain()
	if err != nil {
		gw.logger.WithFields(log.Fields{
			"error": err.Error(),
			"servicePort":  gw.serviceLh.Number(),
		}).Error("net.Listen() error")
		return err
	}
	gw.ln = ln

	gw.logger.WithFields(log.Fields{"servicePort": gw.serviceLh.Number()}).Info("TCP net listener initialized")

	server := grpc.NewServer(grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	for _, handlerFunc := range gw.servicehandlerFuncs {
		handlerFunc(server)
	}
	gw.server = server
	gw.grpcAwaiter = make(chan error)

	go func() {
		gw.logger.Infof("Serving gRPC on :%d", gw.serviceLh.Number())
		err := gw.server.Serve(ln)
		gw.grpcAwaiter <- err
		if err != nil {
			gw.logger.WithFields(log.Fields{"error": err.Error()}).Error("gRPC serve() error")
		}
	}()

	// Starting proxy server
	proxyEndpoint := ":" + strconv.Itoa(gw.proxyPort)
	mux, err := gw.proxyhandlerFunc(proxyEndpoint)
	gw.proxy = &http.Server{Addr: proxyEndpoint, Handler: mux}

	if err != nil {
		gw.logger.WithFields(log.Fields{
			"error":     err.Error(),
			"proxyPort": gw.proxyPort,
		}).Error("RegisterHandlerFromEndpoint() error")
		return err
	}

	go func() {
		gw.logger.Infof("Serving proxy on :%d", gw.proxyPort)
		err := gw.proxy.ListenAndServe()
		gw.grpcAwaiter <- err
		if err != nil {
			gw.logger.WithFields(log.Fields{"error": err.Error()}).Error("proxy ListenAndServe() error")
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
	gw.proxy.Shutdown(context.TODO())
	portErr := gw.ln.Close()
	grpcErr := gw.WaitForTermination()
	gw.server = nil
	gw.ln = nil
	if grpcErr != nil {
		return grpcErr
	}
	return portErr
}

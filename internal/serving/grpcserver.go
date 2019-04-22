package serving

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/GoogleCloudPlatform/open-match/internal/util/netlistener"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// GrpcWrapper is a decoration around the standard GRPC server that sets up a bunch of things common to Open Match servers.
type GrpcWrapper struct {
	serviceLh, proxyLh        *netlistener.ListenerHolder
	serviceHandlerFuncs       []func(*grpc.Server)
	proxyHandlerFunc          func(proxyEndpoint string) (*runtime.ServeMux, error)
	server                    *grpc.Server
	proxy                     *http.Server
	serviceLn, proxyLn        net.Listener
	logger                    *log.Entry
	grpcAwaiter, proxyAwaiter chan error
}

// NewGrpcServer creates a new GrpcWrapper.
func NewGrpcServer(serviceLh, proxyLh *netlistener.ListenerHolder, logger *log.Entry) *GrpcWrapper {
	return &GrpcWrapper{
		serviceLh:           serviceLh,
		proxyLh:             proxyLh,
		logger:              logger,
		serviceHandlerFuncs: []func(*grpc.Server){},
	}
}

// AddService adds a service registration function to be run when the server is created.
func (gw *GrpcWrapper) AddService(handlerFunc func(*grpc.Server)) {
	gw.serviceHandlerFuncs = append(gw.serviceHandlerFuncs, handlerFunc)
}

// AddProxy registers a reverse proxy from REST to gRPC when server is created.
func (gw *GrpcWrapper) AddProxy(proxyHandlerFunc func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error) {
	gw.proxyHandlerFunc = func(endpoint string) (*runtime.ServeMux, error) {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		mux := runtime.NewServeMux()

		type stubPattern struct {
			method string
			ops    []int
			pool   []string
			verb   string
		}

		stub := stubPattern{method: "GET", ops: []int{2, 0}, pool: []string{"ping"}}
		pat, err := runtime.NewPattern(1, stub.ops, stub.pool, stub.verb)

		if err != nil {
			gw.logger.WithFields(log.Fields{
				"error": err.Error(),
				"ops":   stub.ops,
				"pool":  stub.pool,
			}).Error("runtime.NewPattern failed to register /ping")
			return nil, err
		}

		mux.Handle(stub.method, pat, func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
			_, err := io.WriteString(w, "pong")

			if err != nil {
				panic(err)
			}
		})

		return mux, proxyHandlerFunc(ctx, mux, endpoint, []grpc.DialOption{grpc.WithInsecure()})
	}
}

// Start begins the gRPC server.
func (gw *GrpcWrapper) Start() error {
	// Starting gRPC server
	if gw.serviceLn != nil {
		return nil
	}

	serviceLn, err := gw.serviceLh.Obtain()
	if err != nil {
		gw.logger.WithFields(log.Fields{
			"error":       err.Error(),
			"servicePort": gw.serviceLh.Number(),
		}).Error("net.Listen() error")
		return err
	}
	gw.serviceLn = serviceLn

	gw.logger.WithFields(log.Fields{"servicePort": gw.serviceLh.Number()}).Info("TCP net listener initialized")

	server := grpc.NewServer(grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	for _, handlerFunc := range gw.serviceHandlerFuncs {
		handlerFunc(server)
	}
	gw.server = server
	gw.grpcAwaiter = make(chan error)

	go func() {
		gw.logger.Infof("Serving gRPC on :%d", gw.serviceLh.Number())
		err := gw.server.Serve(serviceLn)
		gw.grpcAwaiter <- err
		if err != nil {
			gw.logger.WithFields(log.Fields{"error": err.Error()}).Error("gRPC serve() error")
		}
	}()

	// Starting proxy server
	if gw.proxyLn != nil {
		return nil
	}

	proxyLn, err := gw.proxyLh.Obtain()

	if err != nil {
		gw.logger.WithFields(log.Fields{
			"error":     err.Error(),
			"proxyPort": gw.proxyLh.Number(),
		}).Error("net.Listen() error")
		return err
	}
	gw.proxyLn = proxyLn

	gw.logger.WithFields(log.Fields{"proxyPort": gw.proxyLh.Number()}).Info("TCP net listener initialized")

	serviceEndpoint := fmt.Sprintf("localhost:%d", gw.serviceLh.Number())
	mux, err := gw.proxyHandlerFunc(serviceEndpoint)

	gw.proxy = &http.Server{Handler: mux}
	gw.proxyAwaiter = make(chan error)

	if err != nil {
		gw.logger.WithFields(log.Fields{
			"error":     err.Error(),
			"proxyPort": gw.proxyLh.Number(),
		}).Error("RegisterHandlerFromEndpoint() error")
		return err
	}

	go func() {
		gw.logger.Infof("Serving proxy on :%d", gw.proxyLh.Number())
		err := gw.proxy.Serve(proxyLn)
		gw.proxyAwaiter <- err
		if err != nil {
			gw.logger.WithFields(log.Fields{
				"error":           err.Error(),
				"serviceEndpoint": serviceEndpoint,
				"proxyPort":       gw.proxyLh.Number(),
			}).Error("proxy ListenAndServe() error")
		}
	}()

	return nil
}

// WaitForTermination blocks until the gRPC server is shutdown.
func (gw *GrpcWrapper) WaitForTermination() (error, error) {
	var grpcErr, proxyErr error

	if gw.grpcAwaiter != nil {
		grpcErr = <-gw.grpcAwaiter
		close(gw.grpcAwaiter)
		gw.grpcAwaiter = nil
	}

	if gw.proxyAwaiter != nil {
		proxyErr = <-gw.proxyAwaiter
		close(gw.proxyAwaiter)
		gw.proxyAwaiter = nil
	}
	return grpcErr, proxyErr
}

// Stop gracefully shutsdown the gRPC server.
func (gw *GrpcWrapper) Stop() error {
	if gw.server == nil {
		return nil
	}
	gw.server.GracefulStop()
	gw.proxy.Shutdown(context.Background())

	portErr := gw.serviceLn.Close()
	_ = gw.proxyLn.Close()

	grpcErr, proxyErr := gw.WaitForTermination()

	gw.server = nil
	gw.serviceLn = nil
	gw.proxy = nil
	gw.proxyLn = nil

	if grpcErr != nil {
		return grpcErr
	}
	if proxyErr != nil {
		return proxyErr
	}
	return portErr
}

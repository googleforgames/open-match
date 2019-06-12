package serving

import (
	"github.com/googleforgames/open-match/config"
	"github.com/googleforgames/open-match/internal/logging"
	"github.com/googleforgames/open-match/internal/metrics"
	redishelpers "github.com/googleforgames/open-match/internal/statestorage/redis"
	"github.com/googleforgames/open-match/internal/util/netlistener"
	"github.com/opencensus-integrations/redigo/redis"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
)

// BindingFunc is used as a callback to configure OpenMatchServer most notably the GRPC server instance.
type BindingFunc func(*OpenMatchServer)

// ServerParams is a collection of parameters used to create an Open Match server.
type ServerParams struct {
	BaseLogFields         log.Fields
	ServicePortConfigName string
	ProxyPortConfigName   string
	CustomMeasureViews    []*view.View
	Bindings              []BindingFunc
}

// MustNew panics if an OpenMatchServer cannot be created.
func MustNew(params *ServerParams) *OpenMatchServer {
	srv, err := New(params)
	if err != nil {
		panic(err)
	}
	return srv
}

// New creates an OpenMatchServer based on the parameters.
func New(params *ServerParams) (*OpenMatchServer, error) {
	return NewMulti([]*ServerParams{params})
}

// NewMulti creates an OpenMatchServer based on the parameters.
func NewMulti(paramsList []*ServerParams) (*OpenMatchServer, error) {
	// FIXME: We only take the first item in the list.
	logger := log.WithFields(paramsList[0].BaseLogFields)

	cfg, err := config.Read()
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to load config file")
		return nil, err
	}

	// Configure open match logging defaults
	logging.ConfigureLogging(cfg)

	// Configure OpenCensus exporter to Prometheus
	// metrics.ConfigureOpenCensusPrometheusExporter expects that every OpenCensus view you
	// want to register is in an array, so append any views you want from other
	// packages to a single array here.
	ocServerViews := []*view.View{}
	for _, params := range paramsList {
		ocServerViews = append(ocServerViews, params.CustomMeasureViews...)
	}
	ocServerViews = append(ocServerViews, ocgrpc.DefaultServerViews...)      // gRPC OpenCensus views.
	ocServerViews = append(ocServerViews, config.CfgVarCountView)            // config loader view.
	ocServerViews = append(ocServerViews, redis.ObservabilityMetricViews...) // redis OpenCensus views.
	logger.WithFields(log.Fields{"viewscount": len(ocServerViews)}).Info("Loaded OpenCensus views")
	promLh, err := netlistener.NewFromPortNumber(cfg.GetInt("metrics.port"))
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to create metrics TCP listener")
		return nil, err
	}
	metrics.ConfigureOpenCensusPrometheusExporter(promLh, cfg, ocServerViews)

	// Connect to redis
	pool, err := redishelpers.ConnectionPool(cfg)
	if err != nil {
		logger.Fatal(err)
		return nil, err
	}

	// Instantiate the gRPC server with the bindings we've made.
	grpcLh, err := netlistener.NewFromPortNumber(cfg.GetInt(paramsList[0].ServicePortConfigName))
	if err != nil {
		logger.Fatal(err)
		return nil, err
	}

	proxyLh, err := netlistener.NewFromPortNumber(cfg.GetInt(paramsList[0].ProxyPortConfigName))
	if err != nil {
		logger.Fatal(err)
		return nil, err
	}

	grpcServer := NewGrpcServer(grpcLh, proxyLh, logger)

	omServer := &OpenMatchServer{
		GrpcServer: grpcServer,
		Logger:     logger,
		RedisPool:  pool,
		Config:     cfg,
	}
	for _, params := range paramsList {
		for _, f := range params.Bindings {
			f(omServer)
		}
	}
	return omServer, nil
}

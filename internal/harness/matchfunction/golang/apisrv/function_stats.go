/*
Copyright 2018 Google LLC

/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package apisrv

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// OpenCensus Measures. These are exported as metrics to your monitoring system
// https://godoc.org/go.opencensus.io/stats
//
// When making opencensus stats, the 'name' param, with forward slashes changed
// to underscores, is appended to the 'namespace' value passed to the
// prometheus exporter to become the Prometheus metric name. You can also look
// into having Prometheus rewrite your metric names on scrape.
//
//  For example:
//   - defining the promethus export namespace "open_match" when instanciating the exporter:
//			pe, err := promethus.NewExporter(promethus.Options{Namespace: "openmatch"})
//     (This is in the internal/metrics module)
//   - and naming the request counter view "mmlogic/requests_total":
//			FnRequestCounterView = &view.View{
//				Name:        "mmlogic/requests_total"
//   - results in the prometheus metric name:
//			openmatch_mmlogic_requests_total
//   - [note] when using opencensus views to aggregate the metrics into
//     distribution buckets and such, multiple metrics
//     will be generated with appended types ("<metric>_bucket",
//     "<metric>_count", "<metric>_sum", for example)
//
// In addition, OpenCensus stats propogated to Prometheus have the following
// auto-populated labels pulled from kubernetes, which we should avoid to
// prevent overloading and having to use the HonorLabels param in Prometheus.
//
// - Information about the k8s pod being monitored:
//		"pod" (name of the monitored k8s pod)
//		"namespace" (k8s namespace of the monitored pod)
// - Information about how promethus is gathering the metrics:
//		"instance" (IP and port number being scraped by prometheus)
//		"job" (name of the k8s service being scraped by prometheus)
//		"endpoint" (name of the k8s port in the k8s service being scraped by prometheus)
//
var (
	// Logging instrumentation
	// There's no need to record this measurement directly if you use
	// the logrus hook provided in metrics/helper.go after instantiating the
	// logrus instance in your application code.
	// https://godoc.org/github.com/sirupsen/logrus#LevelHooks
	FnLogLines = stats.Int64("matchfunction/logs_total", "Number of matchfunction lines logged", "1")

	// Matchfunction run instrumentation (separate from gRPC api instrumentation for now)
	FnRequests = stats.Int64("matchfunction/request_total", "Number of matchfunction requests", "1")
	FnFailures = stats.Int64("matchfunction/failures_total", "Number of matchfunction failures", "1")
)

var (
	// KeyMethod is used to tag a measure with the currently running API method.
	KeyMethod, _   = tag.NewKey("method")
	KeyFnName, _   = tag.NewKey("fnname")
	KeySeverity, _ = tag.NewKey("severity")
)

var (
	// Latency in buckets:
	// [>=0ms, >=25ms, >=50ms, >=75ms, >=100ms, >=200ms, >=400ms, >=600ms, >=800ms, >=1s, >=2s, >=4s, >=6s]
	latencyDistribution     = view.Distribution(0, 25, 50, 75, 100, 200, 400, 600, 800, 1000, 2000, 4000, 6000)
	latencyDistributionSecs = view.Distribution(0.0, 0.025, 0.050, 0.075, 0.100, 0.200, 0.400, 0.6, 0.8, 1.0, 2.0, 4.0, 6.0)
)

// Package metrics provides some convience views.
// You need to register the views for the data to actually be collected.
// Note: The OpenCensus View 'Description' is exported to Prometheus as the HELP string.
// Note: If you get a "Failed to export to Prometheus: inconsistent label
// cardinality" error, chances are you forgot to set the tags specified in the
// view for a given measure when you tried to do a stats.Record()
var (
	FnLogCountView = &view.View{
		Name:        "log_lines/total",
		Measure:     FnLogLines,
		Description: "The number of lines logged",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeySeverity, KeyFnName},
	}

	FnRequestsCountView = &view.View{
		Name:        "matchfunction/requests",
		Measure:     FnRequests,
		Description: "The number of requests",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyFnName},
	}

	FnFailureCountView = &view.View{
		Name:        "matchfunction/failures",
		Measure:     FnFailures,
		Description: "The number of failures",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyFnName},
	}
)

// DefaultFunctionAPIViews are the default mmf API OpenCensus measure views.
var DefaultFunctionViews = []*view.View{
	FnLogCountView,
	FnRequestsCountView,
	FnFailureCountView,
}

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
	HarnessLogLines = stats.Int64("harness/logs_total", "Number of harness lines logged", "1")

	// Instrumentation for the harness 'Run' method.
	HarnessRequests   = stats.Int64("harness/request_total", "Number of harness run requests", "1")
	HarnessFailures   = stats.Int64("harness/failures_total", "Number of harness run failures", "1")
	HarnessLatencySec = stats.Float64("harness/latency_seconds", "Latency in seconds for harness runs", "1")

	// Instrumentation for matchfunction execution.
	FnRequests   = stats.Int64("matchfunction/request_total", "Number of matchfunction requests", "1")
	FnFailures   = stats.Int64("matchfunction/failures_total", "Number of matchfunction failures", "1")
	FnLatencySec = stats.Float64("matchfunction/latency_seconds", "Latency in seconds of matchfunction runs", "1")
)

var (
	// KeyMethod is used to tag a measure with the currently running API method.
	KeyMethod, _   = tag.NewKey("method")
	KeyFnName, _   = tag.NewKey("matchfunction")
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
		Measure:     HarnessLogLines,
		Description: "The number of lines logged",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeySeverity, KeyMethod},
	}

	HarnessRequestCountView = &view.View{
		Name:        "harness/requests",
		Measure:     HarnessRequests,
		Description: "The number of successful harness run requests",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyMethod, KeyFnName},
	}

	HarnessFailureCountView = &view.View{
		Name:        "harness/failures",
		Measure:     HarnessFailures,
		Description: "The number of failed harness run requests",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyMethod, KeyFnName},
	}

	HarnessLatencyView = &view.View{
		Name:        "harness/latency",
		Measure:     HarnessLatencySec,
		Description: "The distribution of harness run latencies",
		Aggregation: latencyDistribution,
		TagKeys:     []tag.Key{KeyMethod, KeyFnName},
	}

	FnRequestCountView = &view.View{
		Name:        "matchfunction/requests",
		Measure:     FnRequests,
		Description: "The number of matchfunction requests",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyFnName},
	}

	FnFailureCountView = &view.View{
		Name:        "matchfunction/failures",
		Measure:     FnFailures,
		Description: "The number of matchfunction failures",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyFnName},
	}

	FnLatencyView = &view.View{
		Name:        "matchfunction/latency",
		Measure:     FnLatencySec,
		Description: "The distribution of matchfunction latencies",
		Aggregation: latencyDistributionSecs,
		TagKeys:     []tag.Key{KeyFnName},
	}
)

// DefaultFunctionAPIViews are the default mmf API OpenCensus measure views.
var DefaultFunctionViews = []*view.View{
	FnLogCountView,
	HarnessRequestCountView,
	HarnessFailureCountView,
	HarnessLatencyView,
	FnRequestCountView,
	FnFailureCountView,
	FnLatencyView,
}

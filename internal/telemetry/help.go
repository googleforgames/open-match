// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package telemetry

import (
	"fmt"
	"net/http"
)

const (
	helpEndpoint          = "/help"
	helpSecondaryEndpoint = "/sos"
	helpPage              = `<!DOCTYPE html>
<head>
	<title>Open Match Server Help</title>
</head>
<body>
<pre>
* <a href="/debug/rpcz">/debug/rpcz</a> - RPC Debugging
* <a href="/debug/tracez">/debug/tracez</a> - RPC Tracing
* <a href="/debug/pprof/">/debug/pprof/</a> - PProf
* <a href="/debug/pprof/cmdline">/debug/pprof/cmdline</a> - PProf
* <a href="/debug/pprof/profile">/debug/pprof/profile</a> - PProf
* <a href="/debug/pprof/symbol">/debug/pprof/symbol</a> - PProf
* <a href="/debug/pprof/trace">/debug/pprof/trace</a> - Execution Trace
* <a href="/metrics">/metrics</a> - Raw Metrics, use prometheus or grafana instead.

<i>For /debug/pprof/ links see, https://golang.org/pkg/net/http/pprof/ for details.</i>
</pre>
</body>
`
)

func newHelp() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, helpPage)
	}
}

func bindHelp(p Params, b Bindings) error {
	if !p.Config().GetBool(configNameTelemetryZpagesEnabled) {
		return nil
	}
	h := newHelp()
	b.TelemetryHandleFunc(helpEndpoint, h)
	b.TelemetryHandleFunc(helpSecondaryEndpoint, h)

	return nil
}

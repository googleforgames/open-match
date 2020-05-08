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
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"

	"github.com/spf13/viper"
	"open-match.dev/open-match/internal/config"
)

const (
	configZTemplateName = "configz"
	configEndpoint      = "/configz"
	configPage          = `<!DOCTYPE html>
<head>
	<title>Open Match Configuration</title>
</head>
<body>
<table>
<tr><th>Key</th><th>Value</th></tr>
{{ range $key, $value := . }}
<tr><td>{{ $value.Key }}</td><td>{{ $value.Value }}</td></tr>
{{ end }}
</table>
</body>
`
)

var (
	configPageTemplate = template.Must(template.New(configZTemplateName).Parse(configPage))
)

type configz struct {
	cfg config.View
}

type configZValue struct {
	Key   string
	Value interface{}
}

// ServeHTTP serves the /configz endpoint that allows a user to view the configuration of the server.
func (cz *configz) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	cfg, ok := cz.cfg.(*viper.Viper)
	if !ok {
		http.Error(w, "Configuration is not a *viper.Viper object", http.StatusInternalServerError)
	}
	values := []configZValue{}
	settings := cfg.AllSettings()
	for k, v := range settings {
		values = append(values, configZValue{Key: k, Value: v})
	}
	sort.Slice(values, func(lhs int, rhs int) bool {
		return strings.Compare(values[lhs].Key, values[rhs].Key) != 1
	})
	err := configPageTemplate.Execute(w, values)
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot render HTML template, %s", err), http.StatusInternalServerError)
	}
	var b bytes.Buffer

	err = configPageTemplate.ExecuteTemplate(bufio.NewWriter(&b), configZTemplateName, values)
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot render HTML template, %s", err), http.StatusInternalServerError)
	}
	s := b.String()
	fmt.Print(s)
}

func bindConfigz(p Params, b Bindings) error {
	cfg := p.Config()
	if !cfg.GetBool(configNameTelemetryZpagesEnabled) {
		return nil
	}
	b.TelemetryHandle(configEndpoint, &configz{cfg: cfg})
	return nil
}

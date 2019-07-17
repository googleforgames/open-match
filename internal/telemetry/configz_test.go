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
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
)

func TestConfigz(t *testing.T) {
	assert := assert.New(t)
	cfg := viper.New()
	cfg.Set("char-val", "b")
	cfg.Set("int-val", 1)
	cfg.Set("bool-val", true)
	cz := &configz{cfg: cfg}
	czFunc := func(w http.ResponseWriter, r *http.Request) {
		cz.ServeHTTP(w, r)
	}
	assert.HTTPSuccess(czFunc, http.MethodGet, "/", url.Values{}, "")
	assert.HTTPBodyContains(czFunc, http.MethodGet, "/", url.Values{}, `<!DOCTYPE html>
<head>
	<title>Open Match Configuration</title>
</head>
<body>
<table>
<tr><th>Key</th><th>Value</th></tr>

<tr><td>bool-val</td><td>true</td></tr>

<tr><td>char-val</td><td>b</td></tr>

<tr><td>int-val</td><td>1</td></tr>

</table>
</body>`)
}

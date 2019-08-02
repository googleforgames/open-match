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
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
)

func TestHelp(t *testing.T) {
	assert := assert.New(t)
	h := newHelp()
	assert.HTTPSuccess(h, http.MethodGet, "/", url.Values{}, "")
	assert.HTTPBodyContains(h, http.MethodGet, "/", url.Values{}, `Open Match Server Help`)
}

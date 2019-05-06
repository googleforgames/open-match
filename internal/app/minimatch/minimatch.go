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

// Package minimatch provides in-process serving for all Open Match servers for e2e testing.
package minimatch

import (
	"open-match.dev/open-match/internal/app/backendapi"
	"open-match.dev/open-match/internal/app/frontendapi"
	"open-match.dev/open-match/internal/app/mmlogicapi"
	"open-match.dev/open-match/internal/serving"
)

// CreateServerParams creates the configuration and prepares the binding for serving handler.
func CreateServerParams() []*serving.ServerParams {
	return []*serving.ServerParams{
		frontendapi.CreateServerParams(),
		backendapi.CreateServerParams(),
		mmlogicapi.CreateServerParams(),
	}
}

// RunApplication is a hook for the main() method in the main executable.
func RunApplication() {
	params := CreateServerParams()
	serving.MustServeForeverMulti(params)
}

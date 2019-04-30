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

// Package testing provides test helpers for netlistener.
package testing

import (
	"github.com/GoogleCloudPlatform/open-match/internal/util/netlistener"
)

// MustListen finds the next available port to open for TCP connections, used in tests to make them isolated.
func MustListen() *netlistener.ListenerHolder {
	// Port 0 in Go is a special port number to randomly choose an available port.
	// Reference, https://golang.org/pkg/net/#ListenTCP.
	lh, err := netlistener.NewFromPortNumber(0)
	if err != nil {
		panic(err)
	}
	return lh
}

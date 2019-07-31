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

package util

import (
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "util",
	})
)

// MultiClose is a helper for closing multiple close functions at the end of a program.
type MultiClose struct {
	closers []func()
	m       sync.Mutex
}

// NewMultiClose creates a new multi-closer.
func NewMultiClose() *MultiClose {
	return &MultiClose{
		closers: []func(){},
	}
}

// AddCloseFunc adds a close function.
func (mc *MultiClose) AddCloseFunc(closer func()) {
	mc.m.Lock()
	defer mc.m.Unlock()
	mc.closers = append(mc.closers, closer)
}

// AddCloseWithErrorFunc adds a close function.
func (mc *MultiClose) AddCloseWithErrorFunc(closer func() error) {
	mc.m.Lock()
	defer mc.m.Unlock()
	mc.closers = append(mc.closers, func() {
		err := closer()
		if err != nil {
			logger.WithError(err).Warning("close function failed")
		}
	})
}

// Close shutsdown the server and frees the TCP port.
func (mc *MultiClose) Close() {
	mc.m.Lock()
	defer mc.m.Unlock()
	for _, closer := range mc.closers {
		closer()
	}
	mc.closers = []func(){}
}

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

// Package signal handles terminating applications on Ctrl+Break.
package signal

import (
	"os"
	"os/signal"
)

// New waits for a manual termination or a user initiated termination IE: Ctrl+Break.
// waitForFunc() will wait indefinitely for a signal.
// terminateFunc() will trigger waitForFunc() to complete immediately.
func New() (waitForFunc func(), terminateFunc func()) {
	// Exit when we see a signal
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	waitForFunc = func() {
		<-terminate
	}
	terminateFunc = func() {
		terminate <- os.Interrupt
		close(terminate)
	}
	return waitForFunc, terminateFunc
}

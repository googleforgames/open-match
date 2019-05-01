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

package signal

import (
	"testing"
	"time"
)

const (
	defaultTimeout = time.Duration(100) * time.Millisecond
)

func TestOneSignal(t *testing.T) {
	wait, terminate := New()
	verifyWait(t, wait, terminate)
}

func TestTwoSignals(t *testing.T) {
	wait, terminate := New()
	wait2, terminate2 := New()
	if waitWithTimeout(wait)() {
		t.Error("wait should have timed out because terminate() was not called.")
	}
	if waitWithTimeout(wait2)() {
		t.Error("wait2 should have timed out because terminate2() was not called.")
	}
	terminate()

	if !waitWithTimeout(wait)() {
		t.Error("wait should have completed because terminate() was called.")
	}
	if waitWithTimeout(wait2)() {
		t.Error("wait2 should have timed out because terminate2() was not called.")
	}

	terminate2()
	if !waitWithTimeout(wait)() {
		t.Error("wait should have completed because terminate() was called prior.")
	}
	if !waitWithTimeout(wait2)() {
		t.Error("wait2 should have timed out because terminate2() was called.")
	}
}

func waitWithTimeout(wait func()) func() bool {
	waiter := make(chan struct{})
	go func() {
		defer func() { waiter <- struct{}{} }()
		wait()
	}()

	return func() bool {
		select {
		case <-waiter:
			return true
		case <-time.After(defaultTimeout):
			return false
		}
	}
}

func verifyWait(t *testing.T, wait func(), terminate func()) {
	waiter := make(chan struct{})
	go func() {
		defer func() { waiter <- struct{}{} }()
		wait()
	}()
	terminate()

	select {
	case <-waiter:
		// Success
	case <-time.After(defaultTimeout):
		t.Error("WaitGroup did not complete within 1 second.")
	}
}

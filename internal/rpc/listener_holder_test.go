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

package rpc

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

const (
	numIterations = 1000
)

// TestAddrString verifies that AddrString() is consistent with the port's Addr().String() value.
func TestAddrString(t *testing.T) {
	lh, err := newFromPortNumber(0)
	if err != nil {
		t.Fatalf("newFromPortNumber(0) had error, %s", err)
	}

	if !strings.HasSuffix(lh.AddrString(), fmt.Sprintf(":%d", lh.Number())) {
		t.Errorf("%s does not have suffix ':%d'", lh.AddrString(), lh.Number())
	}

	port, err := lh.Obtain()
	defer func() {
		err = port.Close()
		if err != nil {
			t.Errorf("error %s while calling port.Close()", err)
		}
	}()
	if err != nil {
		t.Errorf("error %s while calling lh.Obtain", err)
	}
	if port.Addr().String() != lh.AddrString() {
		t.Errorf("port.Addr().String() = %s should match lh.AddrString() = %s", port.Addr().String(), lh.AddrString())
	}
}

// TestObtain verifies that a ListenerHolder only returns Obtain() once.
func TestObtain(t *testing.T) {
	var errCount uint64
	var obtainCount uint64
	lh, err := newFromPortNumber(0)
	if err != nil {
		t.Fatalf("newFromPortNumber(0) had error, %s", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < numIterations; i++ {
		wg.Add(1)
		go func() {
			listener, err := lh.Obtain()
			if err != nil {
				atomic.AddUint64(&errCount, 1)
			} else if listener != nil {
				atomic.AddUint64(&obtainCount, 1)
			} else {
				t.Error("err and listener were both nil.")
			}
			wg.Done()
		}()
	}
	wg.Wait()
	finalErrCount := atomic.LoadUint64(&errCount)
	finalObtainCount := atomic.LoadUint64(&obtainCount)
	if finalErrCount != numIterations-1 {
		t.Errorf("expected %d errors, got %d", numIterations-1, finalErrCount)
	}
	if finalObtainCount != 1 {
		t.Errorf("expected %d obtains, got %d", 1, finalObtainCount)
	}
}

func TestMustListen(t *testing.T) {
	for i := 0; i < numIterations; i++ {
		testName := fmt.Sprintf("[%d] MustListen", i)
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			lh := MustListen()
			defer lh.Close()
			if lh.Number() <= 0 {
				t.Errorf("Expected %d > 0, port is out of range.", lh.Number())
			}
		})
	}
}

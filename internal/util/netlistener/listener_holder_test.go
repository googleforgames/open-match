package netlistener

import (
	"sync"
	"sync/atomic"
	"testing"
)

const (
	numIterations = 1000
)

// TestObtain verifies that a ListenerHolder only returns Obtain() once.
func TestObtain(t *testing.T) {
	var errCount uint64
	var obtainCount uint64
	lh, err := NewFromPortNumber(0)
	if err != nil {
		t.Fatalf("NewFromPortNumber(0) had error, %s", err)
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

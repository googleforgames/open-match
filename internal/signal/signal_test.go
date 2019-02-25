package signal

import (
	"sync"
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

func TestSignalWithWaitGroup(t *testing.T) {
	var wg sync.WaitGroup
	waiter := make(chan struct{})
	wait, terminate := New()
	wg.Add(1)
	go func() {
		defer wg.Done()
		wait()
	}()
	terminate()

	go func() {
		wg.Wait()
		waiter <- struct{}{}
	}()
	select {
	case <-waiter:
		// Success
	case <-time.After(defaultTimeout):
		t.Errorf("WaitGroup %v did not complete within 1 second.", wg)
	}
}
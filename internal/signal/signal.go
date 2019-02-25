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
package testing

import (
	"github.com/googleforgames/open-match/internal/util/netlistener"
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

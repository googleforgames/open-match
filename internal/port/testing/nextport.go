package testing

import (
	"fmt"
	"net"
	"sync/atomic"

	"github.com/GoogleCloudPlatform/open-match/internal/port"
)

const (
	firstPort   = 10000
	lastPort    = 60000
	maxAttempts = 1000
)

var (
	currentPort = int32(firstPort)
)

// MustPort finds the next available port to open for TCP connections, used in tests to make them isolated.
func MustPort() *port.Port {
	port, err := nextPort()
	if err != nil {
		panic(err)
	}
	return port
}

// NextPort finds the next available port to open for TCP connections, used in tests to make them isolated.
func nextPort() (*port.Port, error) {
	success := false
	attempt := 0
	for !success && attempt <= maxAttempts {
		p := int(atomic.AddInt32(&currentPort, 1))
		if p > lastPort {
			atomic.CompareAndSwapInt32(&currentPort, int32(p), firstPort)
		} else {
			addr := fmt.Sprintf(":%d", p)
			conn, err := net.Listen("tcp", addr)

			if err == nil {
				fmt.Printf("SELECTPORT %d\n", p)
				return port.CreatePort(p, conn), nil
			}
		}
		attempt++
	}
	return nil, fmt.Errorf("cannot find open port exhausted %d attempts", maxAttempts)
}

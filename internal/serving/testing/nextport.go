package testing

import (
	"fmt"
	"net"
	"sync/atomic"
)

const (
	firstPort   = 10000
	lastPort    = 60000
	maxAttempts = 1000
)

var (
	currentPort = int32(firstPort)
)

func MustPort() int {
	port, err := NextPort()
	if err != nil {
		panic(err)
	}
	return port
}

func NextPort() (int, error) {
	success := false
	attempt := 0
	for !success && attempt <= maxAttempts {
		port := int(atomic.AddInt32(&currentPort, 1))
		if port > lastPort {
			atomic.CompareAndSwapInt32(&currentPort, int32(port), firstPort)
		} else {
			addr := fmt.Sprintf(":%d", port)
			conn, err := net.Listen("tcp", addr)

			if err == nil {
				conn.Close()
				return port, nil
			}
		}
		attempt++
	}
	return -1, fmt.Errorf("cannot find open port exhausted %d attempts", maxAttempts)
}

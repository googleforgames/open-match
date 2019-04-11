package testing

import (
	"fmt"
	"testing"
)

const (
	numIterations = 1000
)

func TestNextPort(t *testing.T) {
	for i := 0; i < numIterations; i++ {
		testName := fmt.Sprintf("[%d] OpenWithAddress", i)
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			port, err := NextPort()
			if err != nil {
				t.Errorf("%s had error, %s", testName, err)
			}
			if !(firstPort <= port && port <= lastPort) {
				t.Errorf("Expected %d <= %d <= %d, port is out of range.", firstPort, port, lastPort)
			}
		})
	}
}

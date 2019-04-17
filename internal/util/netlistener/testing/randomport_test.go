package testing

import (
	"fmt"
	"testing"
)

const (
	numIterations = 1000
)

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

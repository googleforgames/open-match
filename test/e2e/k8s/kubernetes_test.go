package k8s

import (
	"testing"
	"time"
)

func TestShouldFail(t *testing.T) {
	time.Sleep(10 * time.Second)
	t.Fatal("Let's fail")
}

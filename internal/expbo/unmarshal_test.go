package expbo

import (
	"math"
	"testing"
	"time"

	"github.com/cenkalti/backoff"
)

func TestUnmarshalExponentialBackOff(t *testing.T) {
	s := "[0.25 30] *1.5 ~0.33 <300"

	b := backoff.NewExponentialBackOff()
	err := UnmarshalExponentialBackOff(s, b)
	if err != nil {
		t.Fatalf(`error umarshaling "%s": %+v`, s, err)
	}

	if b.InitialInterval != 250*time.Millisecond {
		t.Error("unexpected InitialInterval value:", b.InitialInterval)
	}
	if b.MaxInterval != 30*time.Second {
		t.Error("unexpected MaxInterval value:", b.MaxInterval)
	}
	if math.Abs(b.Multiplier-1.5) > 1e-8 {
		t.Error("unexpected Multiplier value:", b.Multiplier)
	}
	if math.Abs(b.RandomizationFactor-0.33) > 1e-8 {
		t.Error("unexpected RandomizationFactor value:", b.RandomizationFactor)
	}
	if b.MaxElapsedTime != 5*time.Minute {
		t.Error("unexpected MaxElapsedTime value:", b.MaxElapsedTime)
	}
}

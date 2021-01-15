// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package bytesub provides the ability for many clients to subscribe to the
// latest value of a byte array.
package bytesub

import (
	"context"
	"errors"
	"testing"
)

// TestFastAndSlow ensures that if a slow subscriber is blocked, faster subscribers
// nor publishers aren't blocked.  It also ensures that values published while slow
// wasn't listening are skipped.
func TestFastAndSlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fast := make(chan string)
	slow := make(chan string)

	s := New()

	go func() {
		s.Subscribe(ctx, chanWriter{fast})
		close(fast)
	}()
	go func() {
		s.Subscribe(ctx, chanWriter{slow})
		close(slow)
	}()

	for _, i := range []string{"0", "1", "2", "3"} {
		s.AnnounceLatest([]byte(i))
		if v := <-fast; v != i {
			t.Errorf("Expected \"%s\", got \"%s\"", i, v)
		}
	}

	for count := 0; true; count++ {
		if v := <-slow; v == "3" {
			if count > 1 {
				t.Error("Expected to receive at most 1 other value on slow before receiving the latest value.")
			}
			break
		}
	}

	cancel()
	_, ok := <-fast
	if ok {
		t.Error("Expected subscribe to return and fast to be closed")
	}
	_, ok = <-slow
	if ok {
		t.Error("Expected subscribe to return and slow to be closed")
	}
}

func TestBadWriter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := New()
	s.AnnounceLatest([]byte{0, 1, 2})
	err := s.Subscribe(ctx, writerFunc(func(b []byte) (int, error) {
		return 1, nil
	}))

	if err != partialWriteError {
		t.Errorf("Expected partialWriteError, got %s", err)
	}
}

func TestErrorReturned(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expected := errors.New("Hello there.")

	s := New()
	s.AnnounceLatest([]byte{0, 1, 2})
	err := s.Subscribe(ctx, writerFunc(func(b []byte) (int, error) {
		return 0, expected
	}))

	if err != expected {
		t.Errorf("Expected returned error, got %s", err)
	}
}

func TestContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := New()
	s.AnnounceLatest([]byte{0, 1, 2})
	err := s.Subscribe(ctx, writerFunc(func(b []byte) (int, error) {
		return len(b), nil
	}))

	if err != context.Canceled {
		t.Errorf("Expected context canceled error, got %s", err)
	}
}

type chanWriter struct {
	c chan string
}

func (cw chanWriter) Write(b []byte) (int, error) {
	cw.c <- string(b)
	return len(b), nil
}

type writerFunc func(b []byte) (int, error)

func (w writerFunc) Write(b []byte) (int, error) {
	return w(b)
}

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
	"io"
	"sync"
)

var partialWriteError = errors.New("ByteSub subscriber didn't consume the whole message.")

type ByteSub struct {
	nextReady chan struct{}
	b         []byte
	r         sync.RWMutex
}

func New() *ByteSub {
	return &ByteSub{
		nextReady: make(chan struct{}),
	}
}

// AnnounceLatest writes b to all of the subscribers, with caveats listed in Subscribe.
func (s *ByteSub) AnnounceLatest(b []byte) {
	s.r.Lock()
	defer s.r.Unlock()
	close(s.nextReady)
	s.nextReady = make(chan struct{})
	s.b = b
}

func (s *ByteSub) get() (b []byte, nextReady chan struct{}) {
	s.r.RLock()
	defer s.r.RUnlock()
	return s.b, s.nextReady
}

// Subscribe writes the latest value in a single call to w.Write.  It does not
// guarantee that all values published to AnnounceLatest will be written.
// However once things catch up, it will write the latest value.  If no values
// have been announced, waits for a value before writing.
func (s *ByteSub) Subscribe(ctx context.Context, w io.Writer) error {
	nextReady := make(chan struct{})
	close(nextReady)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-nextReady:
			var b []byte
			b, nextReady = s.get()
			if b != nil {
				l, err := w.Write(b)
				if err != nil {
					return err
				}
				if l != len(b) {
					return partialWriteError
				}
			}
		}
	}
}

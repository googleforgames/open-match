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

import "testing"

func TestCollapseByteChan(t *testing.T) {
	in := make(chan []byte)
	out := collapseByteChan(in)

	select {
	case <-out:
		t.Fatal("Out recieved without any input")
	default:
	}

	in <- []byte("0")
	if v := string(<-out); v != "0" {
		t.Fatal("Single value in did not produce expected out.")
	}

	in <- []byte("1")
	in <- []byte("2")
	in <- []byte("3")
	in <- []byte("4")
	in <- []byte("5")

	if v := <-out; string(v) != "5" {
		t.Fatal("Multi value in did not produce expected out.")
	}

	in <- []byte("bad")
	in <- []byte("bad")
	in <- []byte("bad")
	close(in)

	if _, ok := <-out; ok {
		// May recieve 1 bad due to timing, but shouldn't recieve 2.
		if _, ok := <-out; ok {
			t.Fatal("Expected closed channel")
		}
	}
}

func TestCollapseByteChanNoWaitingClose(t *testing.T) {
	in := make(chan []byte)
	out := collapseByteChan(in)

	close(in)

	if _, ok := <-out; ok {
		t.Fatal("Expected closed channel")
	}
}

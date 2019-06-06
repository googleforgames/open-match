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
	"io"
	"time"

	"golang.org/x/net/websocket"
)

type ByteSub struct {
	subscribe chan *chanContext
	latest    chan []byte
}

func New() *ByteSub {
	l := &ByteSub{
		subscribe: make(chan *chanContext),
		latest:    make(chan []byte),
	}

	go multiplexByteChan(collapseByteChan(l.latest), l.subscribe)

	return l
}

func (l *ByteSub) AnnounceLatest(b []byte) {
	l.latest <- b
}

func (l *ByteSub) HandleSubscriber(ws *websocket.Conn) {
	ctx, cancel := context.WithCancel(ws.Request().Context())

	go l.handleSubscriber(ctx, cancel, ws)

	<-ctx.Done()
}

func (l *ByteSub) handleSubscriber(ctx context.Context, cancel context.CancelFunc, w io.Writer) {
	updates := make(chan []byte)

	l.subscribe <- &chanContext{
		c:   updates,
		ctx: ctx,
	}
	collapsed := collapseByteChan(updates)

	for b := range collapsed {
		_, err := w.Write(b)
		if err != nil {
			cancel()
			for range collapsed {
			}
			return
		}
	}
}

// Relays from the input channel to the returned channel.  If multiple inputs
// are sent before the output channel accepts the input, only the most recent
// input is sent.  Closing the input channel closes the output channel, dropping
// any unsent inputs.
func collapseByteChan(in <-chan []byte) <-chan []byte {
	out := make(chan []byte)
	go func() {
		for {
			recent, ok := <-in
			if !ok {
				close(out)
				return
			}

		tryToSend:
			for {
				select {
				case recent, ok = <-in:
					if !ok {
						close(out)
						return
					}
				case out <- recent:
					break tryToSend
				}
			}
		}
	}()
	return out
}

type chanContext struct {
	c   chan<- []byte
	ctx context.Context
}

// Sends inputs to all subscribers.  When the subsciber's context is canceled,
// the subscribers channel will be closed.  If in is closed, the subscribers are
// closed.  A new subscriber gets the most recent value.
func multiplexByteChan(in <-chan []byte, subscribe chan *chanContext) {
	s := make(map[*chanContext]struct{})

	var latest []byte

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {

		case <-ticker.C:
			for cc := range s {
				select {
				case <-cc.ctx.Done():
					delete(s, cc)
					close(cc.c)
				default:
				}
			}

		case v, ok := <-in:
			latest = v
			if !ok {
				for cc := range s {
					close(cc.c)
				}
			}

			for cc := range s {
				select {
				case <-cc.ctx.Done():
					delete(s, cc)
					close(cc.c)
				case cc.c <- v:
				}
			}

		case cc := <-subscribe:
			s[cc] = struct{}{}
			select {
			case <-cc.ctx.Done():
				delete(s, cc)
				close(cc.c)
			case cc.c <- latest:
			}
		}
	}
}

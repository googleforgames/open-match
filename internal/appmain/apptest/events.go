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

// Package apptest allows testing of binded services within memory.
package apptest

// type Event struct {
// 	l    sync.Mutex
// 	c    *sync.Cond
// 	done bool
// }

// func NewEvent() *Event {
// 	e := &Event{}
// 	e.c = sync.NewCond(l)
// 	return e
// }

// func (e *Event) Done() {

// }

// func (e *Event) Wait() {

// }

// func (e *Event) Before(t *testing.T) {
// 	e.l.Lock()
// 	defer e.l.Unlock()
// 	if e.done {
// 		t.Fatal("Event has already happened, assert failed")
// 	}
// }

// func (e *Event) After(t *testing.T) {

// }

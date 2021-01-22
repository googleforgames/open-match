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

// Package updater provides the ability for concurrently running demo pieces to
// update a shared json object.
package updater

import (
	"context"
	"encoding/json"
)

// Updater is like a json object, with each field allowed to be updated
// concurrently by a different process.  After processing updates, Updater will
// call a provided method with the json serialized value of all of its fields.
type Updater struct {
	ctx      context.Context
	children map[string]*json.RawMessage
	updates  chan update
	set      SetFunc
}

// SetFunc serializes the value passed in into json and sets the associated field
// to that value.  If nil is passed (BUT NOT a nil value of an interface), the
// field will be removed from the Updater's json object.
type SetFunc func(v interface{})

// New creates an Updater.  Set is called when fields update, using the json
// serialized value of Updater's tree.  All updates after ctx is canceled are
// ignored.
func New(ctx context.Context, set func([]byte)) *Updater {
	f := func(v interface{}) {
		set([]byte(*forceMarshalJson(v)))
	}
	return NewNested(ctx, SetFunc(f))
}

// NewNested creates an updater based on a field in another updater.  This
// allows for grouping of related demo pieces into a single conceptual group.
func NewNested(ctx context.Context, set SetFunc) *Updater {
	u := create(ctx, set)
	go u.start()
	return u
}

func create(ctx context.Context, set SetFunc) *Updater {
	return &Updater{
		ctx:      ctx,
		children: make(map[string]*json.RawMessage),
		updates:  make(chan update),
		set:      set,
	}
}

// ForField returns a function to set the latest value of that demo piece.
func (u *Updater) ForField(field string) SetFunc {
	return SetFunc(func(v interface{}) {
		var r *json.RawMessage
		if v != nil {
			r = forceMarshalJson(v)
		}

		select {
		case <-u.ctx.Done():
		case u.updates <- update{field, r}:
		}
	})
}

func (u *Updater) start() {
	for {
		u.set(u.children)

		select {
		case <-u.ctx.Done():
			u.set(nil)
			return
		case up := <-u.updates:
			if up.value == nil {
				delete(u.children, up.field)
			} else {
				u.children[up.field] = up.value
			}
		}

	applyAllWaitingUpdates:
		for {
			select {
			case up := <-u.updates:
				if up.value == nil {
					delete(u.children, up.field)
				} else {
					u.children[up.field] = up.value
				}
			default:
				break applyAllWaitingUpdates
			}
		}
	}
}

type update struct {
	field string
	value *json.RawMessage
}

// forceMarshalJson is like json.Marshal, but cannot fail.  It will instead
// encode any error encountered into the json object on the field Error.
func forceMarshalJson(v interface{}) *json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		e := struct {
			Error string
		}{
			err.Error(),
		}

		b, err = json.Marshal(e)
		if err != nil {
			b = []byte("{\"Error\":\"There was an error encoding the json message, additional there was an error encoding that error message.\"}")
		}
	}

	r := json.RawMessage(b)
	return &r
}

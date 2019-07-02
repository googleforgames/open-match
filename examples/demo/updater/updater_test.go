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

package updater

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"testing"
)

func TestUpdater(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	latest := make(chan string)

	base := New(ctx, func(b []byte) {
		latest <- string(b)
	})

	l := <-latest
	if l != "{}" {
		t.Errorf("Got %s, expected %s", l, "{}")
	}

	child := NewNested(ctx, base.ForField("Foo"))

	l = <-latest
	if l != "{\"Foo\":{}}" {
		t.Errorf("Got %s, expected %s", l, "{\"Foo\":{}}")
	}

	child.ForField("Bar")(interface{}((*int)(nil)))

	l = <-latest
	if l != "{\"Foo\":{\"Bar\":null}}" {
		t.Errorf("Got %s, expected %s", l, "{\"Foo\":{\"Bar\":null}}")
	}

	child.ForField("Bar")(nil)

	l = <-latest
	if l != "{\"Foo\":{}}" {
		t.Errorf("Got %s, expected %s", l, "{\"Foo\":{}}")
	}
}

// Fully testing the updater's logic is difficult because it combines multiple
// calls.  This test method creates 100 different go routines all trying to
// update a value to force the logic to be invoked.
func TestUpdaterInternal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(100)

	go func() {
		wg.Wait()
		cancel()
	}()

	latest := ""

	set := SetFunc(func(v interface{}) {
		if v != nil {
			latest = string(*forceMarshalJson(v))
		}
	})

	u := create(ctx, set)

	for i := 0; i < 100; i++ {
		set := u.ForField(strconv.Itoa(i))
		go func() {
			set("Hi")
			wg.Done()
		}()
	}

	// Blocking call ensures that canceling the context will clean up the internal go routine.
	u.start()

	expectedMap := make(map[string]string)
	for i := 0; i < 100; i++ {
		expectedMap[strconv.Itoa(i)] = "Hi"
	}
	// Not using forceMashal because it by design hides errors, and is used in the
	// code being tested.
	expectedB, err := json.Marshal(expectedMap)
	if err != nil {
		t.Fatal(err)
	}
	expected := string(expectedB)

	if latest != expected {
		t.Errorf("latest value is wrong.  Expected '%s', got '%s'", expected, latest)
	}
}

var marshalTests = []struct {
	in  interface{}
	out string
}{
	{map[string]int{"hi": 1}, "{\"hi\":1}"},
	{make(chan int), "{\"Error\":\"json: unsupported type: chan int\"}"},
}

func TestForceMarshalJson(t *testing.T) {
	for _, tt := range marshalTests {
		t.Run(tt.out, func(t *testing.T) {
			s := string(*forceMarshalJson(tt.in))
			if s != tt.out {
				t.Errorf("got %s, want %s", s, tt.out)
			}
		})
	}
}

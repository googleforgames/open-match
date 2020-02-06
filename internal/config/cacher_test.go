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

package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/spf13/viper"
)

var getTests = []struct {
	name           string
	firstValue     interface{}
	firstExpected  interface{}
	secondValue    interface{}
	secondExpected interface{}
	getValue       func(cfg View) interface{}
	verifySame     func(a, b interface{}) bool
}{
	{
		name:           "IsSet",
		firstValue:     nil,
		firstExpected:  false,
		secondValue:    "bar",
		secondExpected: true,
		getValue: func(cfg View) interface{} {
			return cfg.IsSet("foo")
		},
	},
	{
		name:           "GetString",
		firstValue:     "bar",
		firstExpected:  "bar",
		secondValue:    "baz",
		secondExpected: "baz",
		getValue: func(cfg View) interface{} {
			return cfg.GetString("foo")
		},
	},
	{
		name:           "GetInt",
		firstValue:     int(1),
		firstExpected:  int(1),
		secondValue:    int(2),
		secondExpected: int(2),
		getValue: func(cfg View) interface{} {
			return cfg.GetInt("foo")
		},
	},
	{
		name:           "GetInt64",
		firstValue:     int64(1),
		firstExpected:  int64(1),
		secondValue:    int64(2),
		secondExpected: int64(2),
		getValue: func(cfg View) interface{} {
			return cfg.GetInt64("foo")
		},
	},
	{
		name:           "GetFloat64",
		firstValue:     float64(1),
		firstExpected:  float64(1),
		secondValue:    float64(2),
		secondExpected: float64(2),
		getValue: func(cfg View) interface{} {
			return cfg.GetFloat64("foo")
		},
	},
	{
		name:           "GetStringSlice",
		firstValue:     []string{"1", "2", "3"},
		firstExpected:  "2",
		secondValue:    []string{"1", "4", "3"},
		secondExpected: "4",
		getValue: func(cfg View) interface{} {
			return cfg.GetStringSlice("foo")[1]
		},
	},
	{
		name:           "GetStringSliceFirstShorter",
		firstValue:     []string{"1"},
		firstExpected:  1,
		secondValue:    []string{"1", "4", "3"},
		secondExpected: 3,
		getValue: func(cfg View) interface{} {
			return len(cfg.GetStringSlice("foo"))
		},
	},
	{
		name:           "GetStringSliceSecondShorter",
		firstValue:     []string{"1", "2"},
		firstExpected:  2,
		secondValue:    []string{"1"},
		secondExpected: 1,
		getValue: func(cfg View) interface{} {
			return len(cfg.GetStringSlice("foo"))
		},
	},
	{
		name:           "GetBool",
		firstValue:     true,
		firstExpected:  true,
		secondValue:    false,
		secondExpected: false,
		getValue: func(cfg View) interface{} {
			return cfg.GetBool("foo")
		},
	},
	{
		name:           "GetDuration",
		firstValue:     time.Second,
		firstExpected:  time.Second,
		secondValue:    time.Minute,
		secondExpected: time.Minute,
		getValue: func(cfg View) interface{} {
			return cfg.GetDuration("foo")
		},
	},
}

//nolint: gocritic, staticcheck
func Test_Get(t *testing.T) {
	for _, tt := range getTests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.verifySame == nil {
				tt.verifySame = func(a, b interface{}) bool {
					return a == b
				}
			}

			if tt.firstExpected == tt.secondExpected {
				t.Errorf("Expected that first value and second expected would be not equal")
			}

			if tt.firstExpected != tt.firstExpected {
				t.Errorf("Expected that first value would be equal with itself")
			}

			if tt.secondExpected != tt.secondExpected {
				t.Errorf("Expected that first value would be equal with itself")
			}

			cfg := viper.New()
			calls := 0
			var closed interface{}

			cfg.Set("foo", tt.firstValue)
			c := NewCacher(cfg, func(cfg View) (interface{}, func(), error) {
				calls++
				v := tt.getValue(cfg)

				close := func() {
					if closed != nil {
						t.Errorf("Close called without closed being reset.")
					}
					closed = v
				}

				return v, close, nil
			})

			v, err := c.Get()
			if v != tt.firstExpected {
				t.Errorf("expected %v, got %v", tt.firstExpected, v)
			}
			if calls != 1 {
				t.Errorf("expected 1 call, got %d", calls)
			}
			if err != nil {
				t.Errorf("expected nil error, got %v", err)
			}
			if closed != nil {
				t.Errorf("expected nothing closed, got %v", closed)
			}

			cfg.Set("foo", tt.firstValue)

			v, err = c.Get()
			if v != tt.firstExpected {
				t.Errorf("expected %v, got %v", tt.firstExpected, v)
			}
			if calls != 1 {
				t.Errorf("expected 1 call, got %d", calls)
			}
			if err != nil {
				t.Errorf("expected nil error, got %v", err)
			}
			if closed != nil {
				t.Errorf("expected nothing closed, got %v", closed)
			}

			cfg.Set("foo", tt.secondValue)

			v, err = c.Get()
			if v != tt.secondExpected {
				t.Errorf("expected %v, got %v", tt.secondExpected, v)
			}
			if calls != 2 {
				t.Errorf("expected 2 calls, got %d", calls)
			}
			if err != nil {
				t.Errorf("expected nil error, got %v", err)
			}
			if closed != tt.firstExpected {
				t.Errorf("expected first closed, got %v", closed)
			}
			closed = nil
		})
	}
}

func Test_Get_Error(t *testing.T) {
	returnError := true

	cfg := viper.New()
	c := NewCacher(cfg, func(cfg View) (interface{}, func(), error) {
		// Contrived for tests, in real usage outside values shouldn't be used like this.
		if returnError {
			return nil, nil, fmt.Errorf("bad")
		}

		close := func() {
			t.Errorf("Close shouldn't be called.")
		}

		return "foo", close, nil
	})

	v, err := c.Get()

	if v != nil {
		t.Errorf("Expected value to be nil")
	}
	if err.Error() != "bad" {
		t.Errorf("Expected error \"bad\", got %v", err)
	}

	// No config values changed, still trying.
	returnError = false // Emulating the environment changing.
	v, err = c.Get()
	if v != "foo" {
		t.Errorf("Expected foo, got %v", v)
	}
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
}

func Test_ForceReset(t *testing.T) {
	returnValue := "foo"

	var closed interface{}
	cfg := viper.New()
	c := NewCacher(cfg, func(cfg View) (interface{}, func(), error) {

		// Contrived for tests, in real usage outside values shouldn't be used like this.
		v := returnValue

		close := func() {
			if closed != nil {
				t.Errorf("Close called without closed being reset.")
			}
			closed = v
		}

		return v, close, nil
	})

	v, err := c.Get()
	if v != "foo" {
		t.Errorf("Expected foo, got %v", v)
	}
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
	if closed != nil {
		t.Errorf("expected nothing closed, got %v", closed)
	}

	// Environment has changed, eg a server connection has broken and needs to be
	// recreated.  The change is detected with some other means (eg, connection
	// returning errors), and a ForceReset is required.
	returnValue = "bar"

	// Sanity check: value doesn't change because config hasn't.
	v, err = c.Get()
	if v != "foo" {
		t.Errorf("Expected foo, got %v", v)
	}
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
	if closed != nil {
		t.Errorf("expected nothing closed, got %v", closed)
	}

	c.ForceReset()

	// Same thing as above, but ForceReset has been called.
	v, err = c.Get()
	if v != "bar" {
		t.Errorf("Expected bar, got %v", v)
	}
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
	if closed != "foo" {
		t.Errorf("expected foo closed, got %v", closed)
	}
}

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
	"sync"
	"time"
)

// Cacher will detect which config values are read when constructing a value.
// Then, when further requests are made, it will return the same value as long
// as the config values which were used don't change.
type Cacher struct {
	cfg View
	m   sync.Mutex

	r *rememberingView
	v interface{}
}

// NewCacher returns a cacher which uses cfg to detect relevant changes.
func NewCacher(cfg View) *Cacher {
	return &Cacher{
		cfg: cfg,
	}
}

// Get will call f the first time it is called.  It remembers which values are
// read from the provided View, and the value returned.  When Get is next
// called, it will check if any of the used values have changed.  If they
// haven't, it will return the cache'd value.  Otherwise it will call f again.
// If f returns an error, the value will not be cached or returned.
func (c *Cacher) Get(f func(cfg View) (interface{}, error)) (interface{}, error) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.r == nil || c.r.hasChanges() {
		c.r = newRememberingView(c.cfg)
		var err error
		c.v, err = f(c.r)
		if err != nil {
			c.r = nil
			c.v = nil
			return nil, err
		}
	}

	return c.v, nil
}

// ForceReset causes Cacher to forget the remembered value.  The next call to
// Get will again construct a new value.
func (c *Cacher) ForceReset() {
	c.m.Lock()
	defer c.m.Unlock()
	c.r = nil
	c.v = nil
}

type rememberingView struct {
	cfg            View
	isSet          map[string]bool
	getString      map[string]string
	getInt         map[string]int
	getInt64       map[string]int64
	getFloat64     map[string]float64
	getStringSlice map[string][]string
	getBool        map[string]bool
	getDuration    map[string]time.Duration
}

func newRememberingView(cfg View) *rememberingView {
	return &rememberingView{
		cfg:            cfg,
		isSet:          make(map[string]bool),
		getString:      make(map[string]string),
		getInt:         make(map[string]int),
		getInt64:       make(map[string]int64),
		getFloat64:     make(map[string]float64),
		getStringSlice: make(map[string][]string),
		getBool:        make(map[string]bool),
		getDuration:    make(map[string]time.Duration),
	}
}

func (r *rememberingView) IsSet(k string) bool {
	v := r.cfg.IsSet(k)
	r.isSet[k] = v
	return v
}

func (r *rememberingView) GetString(k string) string {
	v := r.cfg.GetString(k)
	r.getString[k] = v
	return v
}

func (r *rememberingView) GetInt(k string) int {
	v := r.cfg.GetInt(k)
	r.getInt[k] = v
	return v
}

func (r *rememberingView) GetInt64(k string) int64 {
	v := r.cfg.GetInt64(k)
	r.getInt64[k] = v
	return v
}

func (r *rememberingView) GetFloat64(k string) float64 {
	v := r.cfg.GetFloat64(k)
	r.getFloat64[k] = v
	return v
}

func (r *rememberingView) GetStringSlice(k string) []string {
	v := r.cfg.GetStringSlice(k)
	r.getStringSlice[k] = v
	return v
}

func (r *rememberingView) GetBool(k string) bool {
	v := r.cfg.GetBool(k)
	r.getBool[k] = v
	return v
}

func (r *rememberingView) GetDuration(k string) time.Duration {
	v := r.cfg.GetDuration(k)
	r.getDuration[k] = v
	return v
}

func (r *rememberingView) hasChanges() bool {
	for k, v := range r.isSet {
		if r.cfg.IsSet(k) != v {
			return true
		}
	}

	for k, v := range r.getString {
		if r.cfg.GetString(k) != v {
			return true
		}
	}

	for k, v := range r.getInt {
		if r.cfg.GetInt(k) != v {
			return true
		}
	}

	for k, v := range r.getInt64 {
		if r.cfg.GetInt64(k) != v {
			return true
		}
	}

	for k, v := range r.getFloat64 {
		if r.cfg.GetFloat64(k) != v {
			return true
		}
	}

	for k, v := range r.getStringSlice {
		actual := r.cfg.GetStringSlice(k)
		if len(actual) != len(v) {
			return true
		}

		for i := range v {
			if v[i] != actual[i] {
				return true
			}
		}
	}

	for k, v := range r.getBool {
		if r.cfg.GetBool(k) != v {
			return true
		}
	}

	for k, v := range r.getDuration {
		if r.cfg.GetDuration(k) != v {
			return true
		}
	}

	return false
}

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

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiCloseNothing(t *testing.T) {
	mc := NewMultiClose()
	mc.Close()
}

func TestMultiCloseSingle(t *testing.T) {
	assert := assert.New(t)
	a := 5
	closeA := func() {
		a++
	}
	mc := NewMultiClose()
	mc.AddCloseFunc(closeA)
	mc.Close()
	assert.Equal(6, a)
}

func TestMultiCloseMulti(t *testing.T) {
	assert := assert.New(t)
	a := 5
	closeA := func() {
		a++
	}
	closeAWithError := func() error {
		a -= 2
		return nil
	}
	b := 5
	closeB := func() {
		b += 10
	}
	mc := NewMultiClose()
	mc.AddCloseFunc(closeA)
	mc.AddCloseWithErrorFunc(closeAWithError)
	mc.AddCloseFunc(closeB)
	mc.Close()
	assert.Equal(4, a)
	assert.Equal(15, b)
}

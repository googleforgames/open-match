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
	"time"

	"github.com/spf13/viper"
)

// View is a read-only view of the Open Match configuration.
// New accessors from Viper should be added here.
type View interface {
	IsSet(string) bool
	GetString(string) string
	GetInt(string) int
	GetInt64(string) int64
	GetFloat64(string) float64
	GetStringSlice(string) []string
	GetBool(string) bool
	GetDuration(string) time.Duration
}

// Mutable is a read-write view of the Open Match configuration.
type Mutable interface {
	Set(string, interface{})
	View
}

// Sub returns a subset of configuration filtered by the key.
func Sub(v View, key string) View {
	vcfg, ok := v.(*viper.Viper)
	if ok {
		return vcfg.Sub(key)
	}
	return nil
}

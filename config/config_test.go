/*
Package config contains convenience functions for reading and managing configuration.

Copyright 2019 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package config

import (
	"testing"
)

func TestReadConfig(t *testing.T) {
	cfg, err := Read()
	if err != nil {
		t.Fatalf("cannot load config, %s", err)
	}

	if cfg.GetString("metrics.endpoint") != "/metrics" {
		t.Errorf("av.GetString('metrics.endpoint') = %s, expected '/metrics'", cfg.GetString("metrics.endpoint"))
	}
}

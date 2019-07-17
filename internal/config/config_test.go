// +build !race

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
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestReadConfigIgnoreRace(t *testing.T) {
	yaml := []byte(`metrics.endpoint: /metrics`)
	if err := ioutil.WriteFile("matchmaker_config.yaml", yaml, 0666); err != nil {
		t.Fatalf("could not create config file: %s", err)
	}
	defer os.Remove("matchmaker_config.yaml")
	yaml = []byte(`metrics.url: om`)
	if err := ioutil.WriteFile("global_config.yaml", yaml, 0666); err != nil {
		t.Fatalf("could not create config file: %s", err)
	}
	defer os.Remove("global_config.yaml")

	cfg, err := Read()
	if err != nil {
		t.Fatalf("cannot load config, %s", err)
	}

	if cfg.GetString("metrics.endpoint") != "/metrics" {
		t.Errorf("av.GetString('metrics.endpoint') = %s, expected '/metrics'", cfg.GetString("metrics.endpoint"))
	}

	yaml = []byte(`metrics.endpoint: ''`)
	if err := ioutil.WriteFile("matchmaker_config.yaml", yaml, 0666); err != nil {
		t.Fatalf("could not update config file: %s", err)
	}

	time.Sleep(2 * time.Second)

	if cfg.GetString("metrics.endpoint") != "" {
		t.Errorf("av.GetString('metrics.endpoint') = %s, expected ''", cfg.GetString("metrics.endpoint"))
	}
}

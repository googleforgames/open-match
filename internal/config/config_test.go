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
	const defaultCfgName = "matchmaker_config_default.yaml"
	const overrideCfgName = "matchmaker_config_override.yaml"

	yaml := []byte(`
metrics.overrideKey: defaultValue1
metrics.defaultKey: defaultValue2
`)
	if err := ioutil.WriteFile(defaultCfgName, yaml, 0666); err != nil {
		t.Fatalf("could not create config file: %s", err)
	}
	defer os.Remove(defaultCfgName)

	yaml = []byte(`metrics.overrideKey: overrideValue1`)
	if err := ioutil.WriteFile(overrideCfgName, yaml, 0666); err != nil {
		t.Fatalf("could not create config file: %s", err)
	}
	defer os.Remove(overrideCfgName)

	cfg, err := Read()
	if err != nil {
		t.Fatalf("cannot load config, %s", err)
	}

	if cfg.GetString("metrics.overrideKey") != "overrideValue1" {
		t.Errorf("cfg.GetString('metrics.overrideKey') = %s, expected 'overrideValue1'", cfg.GetString("metrics.overrideKey"))
	}
	if cfg.GetString("metrics.defaultKey") != "defaultValue2" {
		t.Errorf("cfg.GetString('metrics.defaultKey') = %s, expected 'defaultValue2'", cfg.GetString("metrics.defaultKey"))
	}

	yaml = []byte(`
metrics.newKey: newValue
metrics.overrideKey: overrideValue2
`)
	if err := ioutil.WriteFile(overrideCfgName, yaml, 0666); err != nil {
		t.Fatalf("could not update config file: %s", err)
	}

	time.Sleep(time.Second)

	if cfg.GetString("metrics.overrideKey") != "overrideValue2" {
		t.Errorf("cfg.GetString('metrics.overrideKey') = %s, expected 'overrideValue2'", cfg.GetString("metrics.overrideKey"))
	}
	if cfg.GetString("metrics.defaultKey") != "defaultValue2" {
		t.Errorf("cfg.GetString('metrics.defaultKey') = %s, expected 'defaultValue2'", cfg.GetString("metrics.defaultKey"))
	}
	if cfg.GetString("metrics.newKey") != "newValue" {
		t.Errorf("cfg.GetString('metrics.newKey') = %s, expected 'newValue'", cfg.GetString("metrics.newKey"))
	}
}

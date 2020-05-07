// Copyright 2018 Google LLC
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

// Package config contains convenience functions for reading and managing viper configs.
package config

import (
	"fmt"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Read sets default to a viper instance and read user config to override these defaults.
func Read() (*viper.Viper, error) {
	var err error
	// read configs from config/default/matchmaker_config_default.yaml
	// matchmaker_config_default provides default values for all of the possible tunnable parameters in Open Match
	dcfg := viper.New()
	dcfg.SetConfigType("yaml")
	dcfg.AddConfigPath(".")
	// The config path needs to be the same as the volumeMount path defined via helm
	dcfg.AddConfigPath("/app/config/default")
	dcfg.SetConfigName("matchmaker_config_default")
	err = dcfg.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("fatal error reading default config file, desc: %s", err.Error())
	}

	// read configs from config/override/matchmaker_config_default.yaml
	// matchmaker_config_override overrides default values specified in matchmaker_config_default
	cfg := viper.New()

	// set defaults for cfg using settings in dcfg
	for k, v := range dcfg.AllSettings() {
		cfg.SetDefault(k, v)
	}

	cfg.SetConfigType("yaml")
	cfg.AddConfigPath(".")
	// The config path needs to be the same as the volumeMountPath defined via helm
	cfg.AddConfigPath("/app/config/override")
	cfg.SetConfigName("matchmaker_config_override")
	err = cfg.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("fatal error reading override config file, desc: %s", err.Error())
	}

	// Look for updates to the config; in Kubernetes, this is implemented using
	// a ConfigMap that is written to the matchmaker_config_override.yaml file, which is
	// what the Open Match components using Viper monitor for changes.
	// More details about Open Match's use of Kubernetes ConfigMaps at:
	// https://open-match.dev/open-match/issues/42
	cfg.WatchConfig() // Watch and re-read config file.
	// Write a log when the configuration changes.
	cfg.OnConfigChange(func(event fsnotify.Event) {
		log.Printf("Server configuration changed, operation: %v, filename: %s", event.Op, event.Name)
	})
	return cfg, nil
}

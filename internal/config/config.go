/*
Package config contains convenience functions for reading and managing viper configs.

Copyright 2018 Google LLC

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
	"errors"
	"fmt"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

var (
	// Logrus structured logging setup
	logFields = log.Fields{
		"app":       "openmatch",
		"component": "config",
	}
	cfgLog = log.WithFields(logFields)

	// Map of the config file keys to environment variable names populated by
	// k8s into pods. Examples of redis-related env vars as written by k8s
	// REDIS_SENTINEL_PORT_6379_TCP=tcp://10.55.253.195:6379
	// REDIS_SENTINEL_PORT=tcp://10.55.253.195:6379
	// REDIS_SENTINEL_PORT_6379_TCP_ADDR=10.55.253.195
	// REDIS_SENTINEL_SERVICE_PORT=6379
	// REDIS_SENTINEL_PORT_6379_TCP_PORT=6379
	// REDIS_SENTINEL_PORT_6379_TCP_PROTO=tcp
	// REDIS_SENTINEL_SERVICE_HOST=10.55.253.195
	//
	// MMFs are expected to get their configuation from env vars instead
	// of reading the config file.  So, config parameters that are required
	// by MMFs should be populated to env vars.
	envMappings = map[string]string{
		"redis.user":             "REDIS_USER",
		"redis.password":         "REDIS_PASSWORD",
		"redis.hostname":         "REDIS_SERVICE_HOST",
		"redis.port":             "REDIS_SERVICE_PORT",
		"redis.pool.maxIdle":     "REDIS_POOL_MAXIDLE",
		"redis.pool.maxActive":   "REDIS_POOL_MAXACTIVE",
		"redis.pool.idleTimeout": "REDIS_POOL_IDLETIMEOUT",
		"api.mmlogic.hostname":   "OM_MMLOGICAPI_SERVICE_HOST",
		"api.mmlogic.port":       "OM_MMLOGICAPI_SERVICE_PORT",
	}

	// OpenCensus
	cfgVarCount = stats.Int64("config/vars_total", "Number of config vars read during initialization", "1")
	// CfgVarCountView is the Open Census view for the cfgVarCount measure.
	CfgVarCountView = &view.View{
		Name:        "config/vars_total",
		Measure:     cfgVarCount,
		Description: "The number of config vars read during initialization",
		Aggregation: view.Count(),
	}
)

// Read reads a config file into a viper.Viper instance and associates environment vars defined in
// config.envMappings
func Read(file string) (View, error) {
	cfg := viper.New()
	cfg.SetConfigFile(file)
	err := cfg.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("cannot read config from file %q: %v", file, err.Error())
	}

	// Bind this envvars to viper config vars.
	// https://github.com/spf13/viper#working-with-environment-variables
	// One important thing to recognize when working with ENV variables is
	// that the value will be read each time it is accessed. Viper does not
	// fix the value when the BindEnv is called.
	for cfgKey, envVar := range envMappings {
		err = cfg.BindEnv(cfgKey, envVar)

		if err != nil {
			cfgLog.WithFields(log.Fields{
				"configkey": cfgKey,
				"envvar":    envVar,
				"error":     err.Error(),
				"module":    "config",
			}).Warn("Unable to bind environment var as a config variable")

		} else {
			cfgLog.WithFields(log.Fields{
				"configkey": cfgKey,
				"envvar":    envVar,
				"module":    "config",
			}).Info("Binding environment var as a config variable")
		}
	}

	// Look for updates to the config; in Kubernetes, this is implemented using
	// a ConfigMap that is written to the matchmaker_config.yaml file, which is
	// what the Open Match components using Viper monitor for changes.
	// More details about Open Match's use of Kubernetes ConfigMaps at:
	// https://github.com/GoogleCloudPlatform/open-match/issues/42

	cfg.WatchConfig() // Watch and re-read config file.

	// Write a log when the configuration changes.
	cfg.OnConfigChange(func(event fsnotify.Event) {
		cfgLog.WithFields(log.Fields{
			"filename":  event.Name,
			"operation": event.Op,
		}).Info("Configuration changed.")
	})

	return cfg, nil
}

// ReadAndMerge reads configurations from all specified files using Read(),
// and then merges them into a single viper.Viper instance.
//
// WARNING It doesn't watch for changes currently
func ReadAndMerge(files ...string) (View, error) {
	if len(files) == 0 {
		return nil, errors.New("no input files specified")
	}

	layers := make([]View, len(files))
	for i, f := range files {
		l, err := Read(f)
		if err != nil {
			return nil, err
		}
		layers[i] = l
	}

	cfg := viper.New()
	for _, l := range layers {
		m := l.AllSettings()
		cfg.MergeConfigMap(m)
	}

	// TODO watch layers' changes and re-merge

	return cfg, nil
}

// ReadComponentConfig reads typical configuration for core Open-match component
func ReadComponentConfig() (View, error) {
	return ReadAndMerge(
		// Layer 1: 'per-component' configuration.
		// File is expected to be mounted from ConfigMap.
		"/config/component_config.yaml",

		// Layer 2: configuration that is shared by all OM components.
		// File is expected to be mounted from ConfigMap.
		// This config may also override the settings from previous layer,
		// however all OM components will see it in such case.
		"/config/openmatch_config.yaml",

		// Layer 3: 'expert' configuration that is unlikely to require customizations.
		// Merging of this configuration should be a last step,
		// and by default Open-match does not make any mounts to this path:
		"/config/openmatch_constants.yaml")
}

// ReadSharedConfig reads configuration that is expected to go to all components by default
func ReadSharedConfig() (View, error) {
	return ReadAndMerge(
		"/config/openmatch_config.yaml",
		"/config/openmatch_constants.yaml")
}

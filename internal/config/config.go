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
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "config",
	})

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
		"redis.user":                    "REDIS_USER",
		"redis.password":                "REDIS_PASSWORD",
		"redis.hostname":                "REDIS_SERVICE_HOST",
		"redis.port":                    "REDIS_SERVICE_PORT",
		"redis.pool.maxIdle":            "REDIS_POOL_MAXIDLE",
		"redis.pool.maxActive":          "REDIS_POOL_MAXACTIVE",
		"redis.pool.idleTimeout":        "REDIS_POOL_IDLETIMEOUT",
		"redis.pool.healthCheckTimeout": "REDIS_POOL_HEALTHCHECKTIMEOUT",
		"api.mmlogic.hostname":          "OM_MMLOGICAPI_SERVICE_HOST",
		"api.mmlogic.port":              "OM_MMLOGICAPI_SERVICE_PORT",
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
func Read() (View, error) {
	cfg := viper.New()
	// Viper config management initialization
	// Support either json or yaml file types (json for backwards compatibility
	// with previous versions)
	cfg.SetConfigType("json")
	cfg.SetConfigType("yaml")
	cfg.SetConfigName("matchmaker_config")
	cfg.AddConfigPath(".")
	cfg.AddConfigPath("config")

	// Read in config file using Viper
	err := cfg.ReadInConfig()
	if err != nil {
		logger.WithError(err).Fatal("Fatal error reading config file")
	}

	// Bind this envvars to viper config vars.
	// https://github.com/spf13/viper#working-with-environment-variables
	// One important thing to recognize when working with ENV variables is
	// that the value will be read each time it is accessed. Viper does not
	// fix the value when the BindEnv is called.
	for cfgKey, envVar := range envMappings {
		err = cfg.BindEnv(cfgKey, envVar)

		if err != nil {
			logger.WithError(err).WithFields(logrus.Fields{
				"configkey": cfgKey,
				"envvar":    envVar,
				"module":    "config",
			}).Warn("Unable to bind environment var as a config variable")

		} else {
			logger.WithFields(logrus.Fields{
				"configkey": cfgKey,
				"envvar":    envVar,
				"module":    "config",
			}).Debug("Binding environment var as a config variable")
		}
	}

	// Look for updates to the config; in Kubernetes, this is implemented using
	// a ConfigMap that is written to the matchmaker_config.yaml file, which is
	// what the Open Match components using Viper monitor for changes.
	// More details about Open Match's use of Kubernetes ConfigMaps at:
	// https://open-match.dev/open-match/issues/42
	cfg.WatchConfig() // Watch and re-read config file.
	// Write a log when the configuration changes.
	cfg.OnConfigChange(func(event fsnotify.Event) {
		logger.WithFields(logrus.Fields{
			"filename":  event.Name,
			"operation": event.Op,
		}).Info("Server configuration changed.")
	})
	return cfg, err
}

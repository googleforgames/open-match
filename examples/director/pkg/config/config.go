package config

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	// Logrus structured logging setup
	logFields = log.Fields{
		"app":       "openmatch",
		"component": "director/config",
	}
	cfgLog = log.WithFields(logFields)

	envMappings = map[string]string{
		// TODO
	}

	cfg = viper.New()
)

// Read reads a config file into a viper.Viper instance and associates environment vars defined in
// config.envMappings
func Read() (*viper.Viper, error) {

	// Viper config management initialization
	// Support either json or yaml file types (json for backwards compatibility
	// with previous versions)
	cfg.SetConfigType("json")
	cfg.SetConfigType("yaml")
	cfg.SetConfigName("director_config")
	cfg.AddConfigPath(".")

	// Read in config file using Viper
	err := cfg.ReadInConfig()
	if err != nil {
		cfgLog.WithError(err).Fatal("Fatal error reading config file")
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
	return cfg, err
}

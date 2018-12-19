package logging

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// ConfigureLogging sets up open match logging using the logging section of the matchmaker_config.json
// This includes formatting (text[default] or json) and
// logging levels (debug, info [default], warn, error, fatal, panic)
func ConfigureLogging(cfg *viper.Viper) {
	switch cfg.GetString("logging.format") {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	case "text":
	default:
		logrus.SetFormatter(&logrus.TextFormatter{})
	}

	switch cfg.GetString("logging.level") {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Warn("Debug logging level configured. Not recommended for production!")
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	case "panic":
		logrus.SetLevel(logrus.PanicLevel)
	case "info":
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
}

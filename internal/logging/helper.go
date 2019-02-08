package logging

import (
	stackdriver "github.com/TV4/logrus-stackdriver-formatter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// ConfigureLogging sets up open match logrus instance using the logging section of the matchmaker_config.json
//  - log line format (text[default] or json)
//  - min log level to include (debug, info [default], warn, error, fatal, panic)
//  - include source file and line number for every event (false [default], true)
func ConfigureLogging(cfg *viper.Viper) {
	switch cfg.GetString("logging.format") {
	case "stackdriver":
		logrus.SetFormatter(stackdriver.NewFormatter())
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

	switch cfg.GetBool("logging.source") {
	case true:
		logrus.SetReportCaller(true)
	}

}

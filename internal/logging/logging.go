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

// Package logging configures the Logrus logging library.
package logging

import (
	stackdriver "github.com/TV4/logrus-stackdriver-formatter"
	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/config"
)

// ConfigureLogging sets up open match logrus instance using the logging section of the matchmaker_config.json
//  - log line format (text[default] or json)
//  - min log level to include (debug, info [default], warn, error, fatal, panic)
//  - include source file and line number for every event (false [default], true)
func ConfigureLogging(cfg config.View) {
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
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
		logrus.Warn("Trace logging level configured. Not recommended for production!")
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

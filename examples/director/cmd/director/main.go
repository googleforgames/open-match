package main

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/GoogleCloudPlatform/open-match/internal/logging"

	"github.com/GoogleCloudPlatform/open-match/examples/director/pkg/agones"
	"github.com/GoogleCloudPlatform/open-match/examples/director/pkg/config"
)

var (
	// Profiles debugging config:
	maxSends          int
	maxMatchesPerSend int
	sleepBetweenSends = 30 * time.Second
	complementMatches = true

	allocator Allocator

	cfg = viper.New()

	// Logrus structured logging setup
	dirLogFields = log.Fields{
		"app":       "openmatch",
		"component": "director",
	}
	dirLog = log.WithFields(dirLogFields)

	err = errors.New("")
)

func init() {
	// Viper config management initialization
	cfg, err = config.Read()
	if err != nil {
		dirLog.WithError(err).Fatal("Unable to load config file")
	}

	// Configure open match logging defaults
	logging.ConfigureLogging(cfg)

	dirLog.WithField("cfg", cfg.AllSettings()).Info("Configuration provided")

	// Profiles debugging
	maxSends = cfg.GetInt("debug.maxSends")
	maxMatchesPerSend = cfg.GetInt("debug.maxMatchesPerSend")
	sleepBetweenSends = time.Duration(cfg.GetInt64("debug.sleepBetweenSendsSeconds") * int64(time.Second))

	if cfg.IsSet("debug.complementMatches") {
		complementMatches = cfg.GetBool("debug.complementMatches")
	}

	// Agones
	var namespace, fleetName, generateName string
	if namespace = cfg.GetString("agones.namespace"); namespace == "" {
		dirLog.Fatalf("Incomplete Agones configuration: missing \"agones.namespace\"")
	}
	if fleetName = cfg.GetString("agones.fleetName"); fleetName == "" {
		dirLog.Fatalf("Incomplete Agones configuration: missing \"agones.fleetName\"")
	}
	if generateName = cfg.GetString("agones.generateName"); generateName == "" {
		dirLog.Fatalf("Incomplete Agones configuration: missing \"agones.generateName\"")
	}
	allocator, err = agones.NewGameServerAllocator(namespace, fleetName, generateName, dirLog)
	if err != nil {
		dirLog.WithError(err).Fatal("Could not create Agones allocator")
	}

	dirLog.WithFields(log.Fields{
		"debug.maxSends":          maxSends,
		"debug.maxMatchesPerSend": maxMatchesPerSend,
		"debug.sleepBetweenSends": sleepBetweenSends,
		"debug.complementMatches": complementMatches,

		"agones.namespace":    namespace,
		"agones.fleetName":    fleetName,
		"agones.generateName": generateName,
	}).Debug("Parameters read from configuration")
}

func main() {
	profile, err := readProfile("profile.json")
	if err != nil {
		dirLog.WithError(err).Fatalf(`error reading file "profile.json": %s`, err.Error())
	}

	startSendProfile(context.Background(), profile, dirLog)
}

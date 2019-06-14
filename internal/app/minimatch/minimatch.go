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

package minimatch

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"open-match.dev/open-match/internal/app/backend"
	"open-match.dev/open-match/internal/app/frontend"
	"open-match.dev/open-match/internal/app/mmlogic"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	minimatchUtil "open-match.dev/open-match/internal/util/minimatch"
)

var (
	minimatchLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "minimatch",
	})
	shouldGenerateTickets bool
	shouldAttributes      []string
)

// RunApplication creates a server.
func RunApplication() {
	cfg, err := config.Read()
	if err != nil {
		minimatchLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}
	p, err := rpc.NewServerParamsFromConfig(cfg, "api.frontend")
	if err != nil {
		minimatchLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot construct server.")
	}

	if err := BindService(p, cfg); err != nil {
		minimatchLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot bind server.")
	}

	if err := newMinimatchCmd(cfg).Execute(); err != nil {
		minimatchLogger.Fatal(err)
		return
	}

	rpc.MustServeForever(p)
}

func newMinimatchCmd(cfg config.View) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "minimatch",
		Short:         "A binary that host open-match components in one single port to facilitate development cycle.",
		SilenceErrors: false,
		Args:          cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if shouldGenerateTickets {
				err := minimatchUtil.TicketGenerator(cfg, shouldAttributes, minimatchLogger)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	addFlags(rootCmd.Flags())
	return rootCmd
}

func addFlags(flags *pflag.FlagSet) {
	flags.BoolVarP(&shouldGenerateTickets, "gen", "g", false, "Turn on/off ticket generator.")
	flags.StringSliceVarP(&shouldAttributes, "attributes", "a", []string{"attribute1", "attribute2"}, "Customized ticket index.")
}

// BindService creates the minimatch service to the server Params.
func BindService(p *rpc.ServerParams, cfg config.View) error {
	if err := backend.BindService(p, cfg); err != nil {
		return err
	}

	if err := frontend.BindService(p, cfg); err != nil {
		return err
	}

	if err := mmlogic.BindService(p, cfg); err != nil {
		return err
	}

	return nil
}

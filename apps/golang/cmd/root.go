/*
 * Copyright 2025 Simon Emms <simon@simonemms.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"os"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newRootCmd() *cobra.Command {
	viper.AutomaticEnv()

	var opts struct {
		LogLevel string
	}

	rootCmd := &cobra.Command{
		Use:           "temporal-codec-server",
		Version:       Version,
		Short:         "Decrypt your Temporal data",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			level, err := zerolog.ParseLevel(opts.LogLevel)
			if err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Error setting the log level",
				}
			}
			zerolog.SetGlobalLevel(level)

			return nil
		},
	}

	viper.SetDefault("log_level", zerolog.InfoLevel.String())
	rootCmd.PersistentFlags().StringVarP(
		&opts.LogLevel, "log-level", "l",
		viper.GetString("log_level"), "Set log level",
	)

	rootCmd.AddCommand(
		newConfigCmd(),
		newRunCmd(),
		newVersionCmd(),
	)

	return rootCmd
}

func Execute() {
	err := newRootCmd().Execute()
	if err != nil {
		os.Exit(gh.HandleFatalError(err))
	}
}

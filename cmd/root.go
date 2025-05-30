/*
Copyright © 2025 Simon Emms <simon@simonemms.com>

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
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootOpts struct {
	LogLevel string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "temporal-codec-server",
	Version: Version,
	Short:   "Decrypt your Temporal data",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		level, err := zerolog.ParseLevel(rootOpts.LogLevel)
		if err != nil {
			return err
		}
		zerolog.SetGlobalLevel(level)

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(
		&rootOpts.LogLevel,
		"log-level",
		"l",
		bindEnv[string]("log-level", zerolog.InfoLevel.String()),
		fmt.Sprintf("log level: %s", "Set log level"),
	)
}

func bindEnv[T any](key string, defaultValue ...any) T {
	envvarName := strings.ReplaceAll(key, "-", "_")
	envvarName = strings.ToUpper(envvarName)

	err := viper.BindEnv(key, envvarName)
	cobra.CheckErr(err)

	for _, val := range defaultValue {
		viper.SetDefault(key, val)
	}

	return viper.Get(key).(T)
}

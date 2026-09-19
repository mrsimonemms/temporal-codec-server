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
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCmd_Subcommands(t *testing.T) {
	cmd := newRootCmd()

	names := map[string]bool{}
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}

	assert.True(t, names["version"])
}

func TestNewRootCmd_Flags(t *testing.T) {
	cmd := newRootCmd()

	assert.NotNil(t, cmd.PersistentFlags().Lookup("log-level"))
}

// runCmd returns the run subcommand of a freshly built root command. The root
// command is what wires viper to the environment, so flag defaults resolved
// from envvars are only observable through it.
func runCmd(t *testing.T) *cobra.Command {
	t.Helper()

	for _, sub := range newRootCmd().Commands() {
		if sub.Name() == "run" {
			return sub
		}
	}

	t.Fatal("run subcommand not found")
	return nil
}

// The run command's --config-path default comes from viper, which newRootCmd
// binds to the environment. Without that binding the containerised deployment
// (which sets CONFIG_PATH and passes no flags) would start with an empty
// config path and fail.
func TestNewRunCmd_ConfigPathFlag(t *testing.T) {
	t.Run("defaults to the CONFIG_PATH envvar", func(t *testing.T) {
		t.Setenv("CONFIG_PATH", "/etc/codec/config.yaml")

		flag := runCmd(t).Flags().Lookup("config-path")

		require.NotNil(t, flag)
		assert.Equal(t, "/etc/codec/config.yaml", flag.DefValue)
	})

	t.Run("has a short form", func(t *testing.T) {
		flag := runCmd(t).Flags().Lookup("config-path")

		require.NotNil(t, flag)
		assert.Equal(t, "c", flag.Shorthand)
	})
}

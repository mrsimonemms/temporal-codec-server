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
	"fmt"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/temporal-codec-server/apps/golang/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newConfigInitCmd() *cobra.Command {
	var opts struct {
		Output string
	}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialise a new config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Init()
			if err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Error initialising config file",
				}
			}

			var b []byte
			switch opts.Output {
			case "json":
				b, err = cfg.ToJSON()
			case "yaml":
				b, err = cfg.ToYAML()
			default:
				return gh.FatalError{
					Msg: "Invalid output type",
				}
			}
			if err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Error converting config",
				}
			}

			fmt.Println(string(b))

			return nil
		},
	}

	viper.SetDefault("output", "yaml")
	cmd.Flags().StringVarP(
		&opts.Output, "output", "o",
		viper.GetString("output"), `Output type - one of "json" or "yaml"`,
	)

	return cmd
}

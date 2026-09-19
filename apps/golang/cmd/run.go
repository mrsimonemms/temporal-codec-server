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

	"github.com/gofiber/fiber/v3"
	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/mrsimonemms/temporal-codec-server/apps/golang/internal/router"
	"github.com/mrsimonemms/temporal-codec-server/apps/golang/pkg/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newRunCmd() *cobra.Command {
	var opts struct {
		ConfigPath string
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the Codec server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(opts.ConfigPath)
			if err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Erroring loading config file",
				}
			}

			if err := cfg.Validate(cmd.Context()); err != nil {
				return gh.FatalError{
					Cause: err,
					Msg:   "Config invalid",
				}
			}

			app := fiber.New(fiber.Config{
				AppName: "temporal-codec-server",
			})
			router.New(app, &router.Config{
				BasicUsername:  cfg.Auth.BasicUsername,
				BasicPassword:  cfg.Auth.BasicPassword,
				CORSAllowCreds: cfg.Server.CORSAllowCreds,
				CORSOrigins:    cfg.Server.CORSOrigins,
				EnableAuth:     !cfg.Server.DisableAuth,
				EnableCORS:     !cfg.Server.DisableCORS,
				EnableSwagger:  !cfg.Server.DisableSwagger,
				Encoders:       cfg.GetEncoders(),
				Version:        Version,
			})

			addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
			log.Info().Str("address", addr).Msg("Starting server")

			return app.Listen(addr, fiber.ListenConfig{
				DisableStartupMessage: true,
			})
		},
	}

	cmd.Flags().StringVarP(
		&opts.ConfigPath, "config-path", "c",
		viper.GetString("config_path"), "Path to config file",
	)

	return cmd
}

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
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mrsimonemms/temporal-codec-server/router"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootOpts struct {
	CORSAllowCreds bool
	CORSOrigins    string
	DisableAuth    bool
	DisableCORS    bool
	DisableSwagger bool
	Host           string
	LogLevel       string
	Pause          time.Duration
	Port           int
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
	Run: func(cmd *cobra.Command, args []string) {
		app := fiber.New(fiber.Config{
			AppName:               "temporal-codec-server",
			DisableStartupMessage: true,
			ErrorHandler:          fiberErrorHandler,
		})

		router.New(app, router.Config{
			CORSAllowCreds: rootOpts.CORSAllowCreds,
			CORSOrigins:    rootOpts.CORSOrigins,
			EnableAuth:     !rootOpts.DisableAuth,
			DisableCORS:    rootOpts.DisableCORS,
			DisableSwagger: rootOpts.DisableSwagger,
			Pause:          rootOpts.Pause,
		})

		addr := fmt.Sprintf("%s:%d", rootOpts.Host, rootOpts.Port)
		log.Info().Str("address", addr).Msg("Starting server")

		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("Error starting server")
		}
	},
}

func fiberErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	var output any
	var e *fiber.Error

	if errors.As(err, &e) {
		code = e.Code
		output = e
	}

	if code >= 500 && code < 600 {
		// Log as developer error
		log.Error().Err(err).Msg("Error")
	} else {
		// Log as human error
		log.Debug().Err(err).Msg(e.Message)
	}

	// Render the error as JSON
	err = c.Status(code).JSON(output)
	if err != nil {
		log.Error().Err(err).Msg("Error rendering web output")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.ErrInternalServerError)
	}

	return nil
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

	rootCmd.Flags().BoolVar(
		&rootOpts.CORSAllowCreds,
		"cors-allow-creds",
		bindEnv[bool]("cors-allow-creds", true),
		"Configure the CORS Access-Control-Allow-Credentials header",
	)
	rootCmd.Flags().StringVar(
		&rootOpts.CORSOrigins,
		"cors-origins",
		bindEnv[string]("cors-origins", "https://cloud.temporal.io"),
		"Configure the CORS Access-Control-Allow-Origin header",
	)
	rootCmd.Flags().BoolVar(&rootOpts.DisableAuth, "disable-auth", bindEnv[bool]("disable-auth", false), "Disable endpoint authorization")
	rootCmd.Flags().BoolVar(&rootOpts.DisableCORS, "disable-cors", bindEnv[bool]("disable-cors", false), "Disable CORS")
	rootCmd.Flags().BoolVar(&rootOpts.DisableSwagger, "disable-swagger", bindEnv[bool]("disable-swagger", false), "Disable Swagger endpoint")
	rootCmd.Flags().StringVarP(&rootOpts.Host, "host", "H", bindEnv[string]("host", ""), "Server listen host")
	rootCmd.Flags().IntVarP(&rootOpts.Port, "port", "p", bindEnv[int]("port", 3000), "Server listen port")
	rootCmd.Flags().DurationVar(
		&rootOpts.Pause,
		"pause",
		bindEnv[time.Duration]("pause", time.Nanosecond*0),
		"Artificial pause before encoding and decoding endpoints are resolved",
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

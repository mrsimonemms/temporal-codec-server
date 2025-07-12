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
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/mrsimonemms/temporal-codec-server/apps/golang/router"
	"github.com/mrsimonemms/temporal-codec-server/apps/golang/views"
	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

var rootOpts struct {
	BasicUsername      string
	BasicPassword      string
	CORSAllowCreds     bool
	CORSOrigins        string
	DisableAuth        bool
	DisableCORS        bool
	DisableSwagger     bool
	EncryptionKeysPath string
	Host               string
	LogLevel           string
	Pause              time.Duration
	Port               int
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
		// Get the encryption keys
		keys, err := aes.ReadKeyFile(rootOpts.EncryptionKeysPath)
		if err != nil {
			log.Fatal().Err(err).Str("file path", rootOpts.EncryptionKeysPath).Msg("Unable to get keys from file")
		}

		// Create the encoders map for each namespace
		encoders := map[string][]converter.PayloadCodec{
			client.DefaultNamespace: {aes.NewPayloadCodec(keys)},
		}

		app := fiber.New(fiber.Config{
			AppName:               "temporal-codec-server",
			DisableStartupMessage: true,
			ErrorHandler:          fiberErrorHandler,
			Views:                 html.NewFileSystem(http.FS(views.ViewsDir), ".html"),
		})

		router.New(app, router.Config{
			BasicUsername:  rootOpts.BasicUsername,
			BasicPassword:  rootOpts.BasicPassword,
			CORSAllowCreds: rootOpts.CORSAllowCreds,
			CORSOrigins:    rootOpts.CORSOrigins,
			EnableAuth:     !rootOpts.DisableAuth,
			EnableCORS:     !rootOpts.DisableCORS,
			EnableSwagger:  !rootOpts.DisableSwagger,
			Encoders:       encoders,
			Pause:          rootOpts.Pause,
			Version:        Version,
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
	viper.AutomaticEnv()

	viper.SetDefault("log_level", zerolog.InfoLevel.String())
	rootCmd.PersistentFlags().StringVarP(
		&rootOpts.LogLevel,
		"log-level",
		"l",
		viper.GetString("log_level"),
		fmt.Sprintf("log level: %s", "Set log level"),
	)

	viper.SetDefault("basic_username", "")
	rootCmd.Flags().StringVar(
		&rootOpts.BasicUsername,
		"basic-username",
		viper.GetString("basic_username"),
		"Add HTTP Basic username for authentication",
	)

	viper.SetDefault("basic_password", "")
	rootCmd.Flags().StringVar(
		&rootOpts.BasicPassword,
		"basic-password",
		viper.GetString("basic_password"),
		"Add HTTP Basic password for authentication",
	)

	viper.SetDefault("cors_allow_creds", true)
	rootCmd.Flags().BoolVar(
		&rootOpts.CORSAllowCreds,
		"cors-allow-creds",
		viper.GetBool("cors_allow_creds"),
		"Configure the CORS Access-Control-Allow-Credentials header",
	)

	viper.SetDefault("cors_origins", "https://cloud.temporal.io")
	rootCmd.Flags().StringVar(
		&rootOpts.CORSOrigins,
		"cors-origins",
		viper.GetString("cors_origins"),
		"Configure the CORS Access-Control-Allow-Origin header. Accepts comma-separated values and can use wildcards",
	)

	viper.SetDefault("disable_auth", false)
	rootCmd.Flags().BoolVar(
		&rootOpts.DisableAuth,
		"disable-auth",
		viper.GetBool("disable_auth"),
		"Disable endpoint authorization",
	)

	viper.SetDefault("disable_cors", false)
	rootCmd.Flags().BoolVar(
		&rootOpts.DisableCORS,
		"disable-cors",
		viper.GetBool("disable_cors"),
		"Disable CORS",
	)

	viper.SetDefault("disable_swagger", false)
	rootCmd.Flags().BoolVar(
		&rootOpts.DisableSwagger,
		"disable-swagger",
		viper.GetBool("disable_swagger"),
		"Disable Swagger endpoint",
	)

	viper.SetDefault("keys_path", "")
	rootCmd.Flags().StringVar(
		&rootOpts.EncryptionKeysPath,
		"keys-path",
		viper.GetString("keys_path"),
		"Path of JSON file for encryption keys in key/value format",
	)

	viper.SetDefault("host", "")
	rootCmd.Flags().StringVarP(&rootOpts.Host, "host", "H", viper.GetString("host"), "Server listen host")

	viper.SetDefault("port", 3000)
	rootCmd.Flags().IntVarP(&rootOpts.Port, "port", "p", viper.GetInt("port"), "Server listen port")

	viper.SetDefault("pause", 0)
	rootCmd.Flags().DurationVar(
		&rootOpts.Pause,
		"pause",
		viper.GetDuration("pause"),
		"Artificial pause before encoding and decoding endpoints are resolved",
	)
}

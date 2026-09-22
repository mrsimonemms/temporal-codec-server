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

//go:generate swag init --output ../docs -g router.go --parseDependency --parseInternal

package router

import (
	"net/http"

	swaggo "github.com/gofiber/contrib/v3/swaggo"
	fiberlogger "github.com/gofiber/contrib/v3/zerolog"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/converter"

	_ "github.com/mrsimonemms/temporal-codec-server/apps/golang/internal/docs"
	"github.com/mrsimonemms/temporal-codec-server/packages/golang/auth"
)

type router struct {
	app *fiber.App
	cfg *Config
}

// @title Temporal Codec Server
// @version 1.0
// @description Decrypt your Temporal data
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /
// @contact.name Simon Emms
// @contact.url https://github.com/mrsimonemms/temporal-codec-server
func (r *router) register() {
	// ################################ //
	// Configure the web app's settings //
	// ################################ //
	r.app.
		// Add a request ID to each HTTP call
		Use(requestid.New()).
		Use(fiberlogger.New(fiberlogger.Config{
			Logger: &log.Logger,
			Fields: []string{"latency", "status", "method", "url", "error"},
			GetLogger: func(c fiber.Ctx) zerolog.Logger {
				return log.With().
					Str("requestid", requestid.FromContext(c)).
					Logger()
			},
		})).
		// Log each endpoint and inject into context
		Use(func(c fiber.Ctx) error {
			l := log.With().
				Str("requestid", requestid.FromContext(c)).
				Str("method", c.Method()).
				Bytes("url", c.Request().URI().Path()). // Avoid logging any sensitive credentials
				Logger()

			c.Locals(loggerKey, l)

			l.Debug().Msg("New route called")

			return c.Next()
		}).
		// Allow recovery from "panic"
		Use(recover.New())

	if r.cfg.EnableCORS {
		// Enable CORS configuration
		log.Debug().
			Bool("allow creds", r.cfg.CORSAllowCreds).
			Strs("origins", r.cfg.CORSOrigins).
			Msg("Enabling CORS")

		r.app.Use(cors.New(cors.Config{
			AllowCredentials: r.cfg.CORSAllowCreds,
			AllowHeaders: []string{
				"Authorization",
				"Content-Type",
				"X-Namespace",
			},
			AllowMethods: []string{
				fiber.MethodGet,
				fiber.MethodPost,
				fiber.MethodOptions,
			},
			AllowOrigins: r.cfg.CORSOrigins,
		}))
	}

	// ################### //
	// Register the routes //
	// ################### //

	if r.cfg.EnableSwagger {
		log.Debug().Msg("Adding Swagger endpoints")
		r.app.Get("api/*", swaggo.HandlerDefault)
	}

	// Webpages
	r.app.Get("/", func(c fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"EnableSwagger": r.cfg.EnableSwagger,
			"Version":       r.cfg.Version,
			"Year":          2025,
		})
	})

	// Health and observability checks
	r.app.Use(healthcheck.LivenessEndpoint, healthcheck.New(healthcheck.Config{
		Probe: r.healthcheckProbe,
	}))
	r.app.Use(healthcheck.ReadinessEndpoint, healthcheck.New(healthcheck.Config{
		Probe: r.healthcheckProbe,
	}))
	r.app.Get("/metrics", r.metrics())

	// Temporal endpoints
	authFns := []auth.MiddlewareAuthFunction{
		auth.TemporalJWKS,
	}
	if r.cfg.BasicUsername != "" && r.cfg.BasicPassword != "" {
		log.Debug().Msg("Add HTTP Basic authentication")
		authFns = append(authFns, auth.HTTPBasic(r.cfg.BasicUsername, r.cfg.BasicPassword))
	}

	handlers := []fiber.Handler{
		// Check if we should enforce authorisation
		r.middlewareAuth(auth.OneOf(authFns...)),
		// Codec converter handler
		r.codecConverter,
	}
	first, rest := toVariadic(handlers)

	r.app.
		Post("/decode", first, rest...).
		Post("/encode", first, rest...).
		Post("/:namespace/decode", first, rest...).
		Post("/:namespace/encode", first, rest...)
}

func toVariadic(handlers []fiber.Handler) (any, []any) {
	rest := make([]any, len(handlers)-1)
	for i, h := range handlers[1:] {
		rest[i] = h
	}
	return handlers[0], rest
}

type Config struct {
	BasicUsername  string
	BasicPassword  string
	CORSAllowCreds bool
	CORSOrigins    []string
	EnableAuth     bool
	EnableCORS     bool
	EnableSwagger  bool
	Encoders       map[string][]converter.PayloadCodec
	Version        string

	codecHandlers map[string]http.Handler
}

func (c *Config) buildCodecHandlers() {
	encoders := c.Encoders

	c.codecHandlers = make(map[string]http.Handler, len(encoders))
	for namespace, codecChain := range encoders {
		log.Debug().Str("namespace", namespace).Msg("Implementing codec handler")

		handler := converter.NewPayloadCodecHTTPHandler(codecChain...)

		c.codecHandlers[namespace] = handler
	}
}

func (c *Config) GetCodecHandlers() map[string]http.Handler {
	return c.codecHandlers
}

func New(app *fiber.App, cfg *Config) *router {
	cfg.buildCodecHandlers()

	r := &router{
		app: app,
		cfg: cfg,
	}

	r.register()

	return r
}

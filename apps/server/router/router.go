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
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/swagger"
	"github.com/rs/zerolog/log"

	_ "github.com/mrsimonemms/temporal-codec-server/apps/server/docs"
)

type router struct {
	app *fiber.App
	cfg Config
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
		// Log each endpoint and inject into context
		Use(func(c *fiber.Ctx) error {
			l := log.With().
				Interface("requestid", c.Locals(requestid.ConfigDefault.ContextKey)).
				Str("method", c.Method()).
				Bytes("url", c.Request().URI().Path()). // Avoid logging any sensitive credentials
				Logger()

			c.Locals("logger", l)

			l.Debug().Msg("New route called")

			return c.Next()
		}).
		// Allow recovery from "panic"
		Use(recover.New())

	// ################### //
	// Register the routes //
	// ################### //

	if r.cfg.EnableSwagger {
		log.Debug().Msg("Adding Swagger endpoints")
		r.app.Get("api/*", swagger.HandlerDefault)
	}

	// Health and observability checks
	r.app.Use(healthcheck.New(healthcheck.Config{
		LivenessProbe:  r.healthcheckProbe,
		ReadinessProbe: r.healthcheckProbe,
	}))
	r.app.Get("/metrics", r.metrics())

	// Temporal endpoints
	r.app.Post("/decode", r.codecDecode)
}

type Config struct {
	EnableSwagger bool
}

func New(app *fiber.App, cfg Config) *router {
	r := &router{
		app: app,
		cfg: cfg,
	}

	r.register()

	return r
}

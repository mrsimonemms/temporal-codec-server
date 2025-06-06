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

package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

type router struct {
	app *fiber.App
}

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

	// Health and observability checks
	r.app.Use(healthcheck.New(healthcheck.Config{
		LivenessProbe:  r.healthcheckProbe,
		ReadinessProbe: r.healthcheckProbe,
	}))
	r.app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
}

func New(app *fiber.App) *router {
	r := &router{
		app: app,
	}

	r.register()

	return r
}

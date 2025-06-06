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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

// Health check godoc
// @Summary		Health check
// @Description Perform a service health check
// @Tags		Health
// @Accept		plain
// @Produce		plain
// @Success		200	"OK"
// @Failure		503 "Service Unavailable"
// @Router		/livez [get]
// @Router		/readyz [get]
func (r *router) healthcheckProbe(c *fiber.Ctx) bool {
	log := c.Locals("logger").(zerolog.Logger)

	log.Debug().Msg("Service healthy")
	return true
}

// Prometheus metrics godoc
// @Summary		Metrics
// @Description Return Prometheus metrics for this service
// @Tags		Health
// @Accept		plain
// @Produce		plain
// @Success		200	{string} OK
// @Router		/metrics [get]
func (r *router) metrics() fiber.Handler {
	return adaptor.HTTPHandler(promhttp.Handler())
}

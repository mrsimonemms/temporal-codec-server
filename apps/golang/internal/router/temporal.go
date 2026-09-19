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
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"go.temporal.io/sdk/client"
)

// Define the types for Swagger
type Payloads struct {
	Payloads []Payload `json:"payloads"`
}

// Define the types for Swagger
type Payload struct {
	Metadata map[string]string `json:"metadata" example:"encoding:YmluYXJ5L3NuYXBweQ=="`
	Data     string            `json:"data" example:"NdAKFgoIZW5jb2RpbmcSCmpzb24vcGxhaW4SGyJSZWNlaXZlZCBQbGFpbiB0ZXh0IGlucHV0Ig=="`
}

// Encode and decode your Temporal data godoc
// @Summary		Encode and decode Temporal data
// @Description Encode and decode your encrypted Temporal data.
// @Tags		Temporal
// @Accept		json
// @Produce		json
// @Success		200	{string} OK
// @Param		namespace	path	string	true	"Temporal namespace"
// @Router		/decode [post]
// @Router		/encode [post]
// @Router		/{namespace}/decode [post]
// @Router		/{namespace}/encode [post]
// @Param		payload	body	Payloads	true	"Encoded payload data"
// @Success		200	{object}	Payloads
func (r *router) codecConverter(c fiber.Ctx) error {
	log := GetLogger(c)
	codecHandlers := r.cfg.GetCodecHandlers()
	namespace := c.Params("namespace")

	if namespace == "" {
		namespace = c.Get("X-Namespace")
	}

	if namespace == "" {
		namespace = client.DefaultNamespace
	}

	handler, ok := codecHandlers[namespace]
	if !ok {
		log.Error().Msg("Unknown namespace")
		return fiber.ErrNotFound
	}

	log.Debug().Str("namespace", namespace).Msg("Executing codec handler")
	return adaptor.HTTPHandler(handler)(c)
}

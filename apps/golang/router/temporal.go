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
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"go.temporal.io/sdk/converter"
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

// Decrypt your Temporal data godoc
// @Summary		Decode Temporal data
// @Description Decrypt your encrypted Temporal data
// @Tags		Temporal
// @Accept		json
// @Produce		json
// @Success		200	{string} OK
// @Router		/decode [post]
// @Param		payload	body	Payloads	true	"Encrypted payload data"Add commentMore actions
// @Success		200	{object}	Payloads
func (r *router) codecDecode(c *fiber.Ctx) error {
	encoders := r.cfg.Encoders

	codecHandlers := make(map[string]http.Handler, len(encoders))
	for namespace, codecChain := range encoders {
		handler := converter.NewPayloadCodecHTTPHandler(codecChain...)

		codecHandlers[namespace] = handler
	}

	defaultHandler, ok := codecHandlers["default"]
	if !ok {
		return fiber.ErrNotFound
	}

	return adaptor.HTTPHandler(defaultHandler)(c)
}

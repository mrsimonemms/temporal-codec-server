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
	"github.com/mrsimonemms/temporal-codec-server/packages/golang"
	"go.temporal.io/sdk/converter"
)

// Decrypt your Temporal data godoc
// @Summary		Decode Temporal data
// @Description Decrypt your encrypted Temporal data
// @Tags		Temporal
// @Accept		json
// @Produce		json
// @Success		200	{string} OK
// @Router		/decode [post]
func (r *router) codecDecode(c *fiber.Ctx) error {
	encoders := map[string][]converter.PayloadCodec{
		"default": {golang.NewPayloadCodec()},
	}

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

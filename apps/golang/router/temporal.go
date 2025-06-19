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
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/client"
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
func (r *router) codecConverter(c *fiber.Ctx) error {
	log := c.Locals("logger").(zerolog.Logger)
	encoders := r.cfg.Encoders

	codecHandlers := make(map[string]http.Handler, len(encoders))
	for namespace, codecChain := range encoders {
		log.Debug().Str("namespace", namespace).Msg("Implementing codec hancler")

		handler := converter.NewPayloadCodecHTTPHandler(codecChain...)

		codecHandlers[namespace] = handler
	}

	// Get the namespace - use the default namespace unless told otherwise
	namespace := client.DefaultNamespace
	if nsp := c.Params("namespace"); nsp != "" {
		// Set by route - this cannot be changed
		log.Debug().Str("namespace", nsp).Msg("Namespace set by route")
		namespace = nsp
	} else {
		// Namespace not set by the route - first look for an x-namespace parameter
		log.Debug().Msg("No namespace set in the route - searching for header")
		namespaceHeader := c.Get("X-Namespace")
		if namespaceHeader != "" {
			// Does the namespace in the header have a configured codec handler?
			log.Debug().Msg("X-Namespace header is set - looking for hander")
			if _, ok := codecHandlers[namespaceHeader]; ok {
				log.Debug().Msg("Using namespace in X-Namespace header")
				namespace = namespaceHeader
			}
		}
	}

	log = log.With().Str("namespace", namespace).Logger()

	log.Debug().Msg("Finding codec handler")
	handler, ok := codecHandlers[namespace]
	if !ok {
		log.Error().Msg("Unknown namespace")
		return fiber.ErrNotFound
	}

	log.Debug().Msg("Executing codec handler")
	return adaptor.HTTPHandler(handler)(c)
}

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
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"go.temporal.io/server/common/authorization"
)

// Conditional function
func MiddlewareAuthConditional(enabled bool) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		log := c.Locals("logger").(zerolog.Logger)

		if !enabled {
			log.Debug().Msg("Authorisation not enabled - passing through")
			return c.Next()
		}

		log.Debug().Msg("Authorisation enabled")
		return MiddlewareAuth(c)
	}
}

// Exportable function
func MiddlewareAuth(c *fiber.Ctx) error {
	log := c.Locals("logger").(zerolog.Logger)

	var authInfo *authorization.AuthInfo

	log.Debug().Msg("Looking for token")
	auth, ok := c.GetReqHeaders()["Authorization"]
	if ok && len(auth) == 1 {
		log.Debug().Msg("Auth header found")
		split := strings.Split(auth[0], "Bearer")
		if len(split) == 2 {
			log.Debug().Msg("Auth header is a bearer token")
			token := strings.TrimSpace(split[1])

			authInfo = &authorization.AuthInfo{
				AuthToken: token,
			}
		}
	}

	fmt.Println(authInfo)

	return fiber.ErrUnauthorized
}

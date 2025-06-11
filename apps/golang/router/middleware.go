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
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

func (r *router) middlewareAddDelay(c *fiber.Ctx) error {
	log := c.Locals("logger").(zerolog.Logger).With().Dur("delay", r.cfg.Pause).Logger()

	if r.cfg.Pause > 0 {
		log.Debug().Msg("Pausing before resolving endpoints")
		time.Sleep(r.cfg.Pause)
		log.Debug().Msg("Pause ending")
	}
	return c.Next()
}

// Ensure that only authorised users can access
func (r *router) middlewareAuth(authFN MiddlewareAuthFunction) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		log := c.Locals("logger").(zerolog.Logger)

		if !r.cfg.EnableAuth {
			log.Debug().Msg("Authorisation not enabled - passing through")
			return c.Next()
		}

		log.Debug().Msg("Looking for token")
		var token string
		authHeader, ok := c.GetReqHeaders()["Authorization"]
		if ok && len(authHeader) == 1 {
			log.Debug().Msg("Auth header found")
			split := strings.Split(authHeader[0], "Bearer")
			if len(split) == 2 {
				log.Debug().Msg("Auth header is a bearer token")
				token = strings.TrimSpace(split[1])
			}
		}

		if token != "" {
			log.Debug().Msg("Validating the auth header")

			if err := authFN(token); err == nil {
				log.Debug().Msg("Token is valid")
				return c.Next()
			}
		}

		log.Debug().Msg("Authentication failed")
		return fiber.ErrUnauthorized
	}
}

type MiddlewareAuthFunction func(string) error

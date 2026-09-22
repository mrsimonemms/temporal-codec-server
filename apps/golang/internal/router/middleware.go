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

	"github.com/gofiber/fiber/v3"
	"github.com/mrsimonemms/temporal-codec-server/packages/golang/auth"
)

// Ensure that only authorised users can access
func (r *router) middlewareAuth(authFN auth.MiddlewareAuthFunction) func(c fiber.Ctx) error {
	return func(c fiber.Ctx) error {
		log := GetLogger(c)

		if strings.HasSuffix(c.Path(), "/encode") {
			log.Debug().Msg("Authorisation not enabled for encode endpoints")
			return c.Next()
		}

		if !r.cfg.EnableAuth {
			log.Debug().Msg("Authorisation not enabled - passing through")
			return c.Next()
		}

		log.Debug().Msg("Looking for token")
		var authType string
		var token string
		authHeader := c.Get("Authorization")
		if authHeader != "" {
			log.Debug().Msg("Auth header found")
			split := strings.Split(authHeader, " ")
			if len(split) == 2 {
				log.Debug().Msg("Auth header is set")
				authType = strings.TrimSpace(split[0])
				token = strings.TrimSpace(split[1])
			}
		}

		if authType != "" && token != "" {
			log.Debug().Msg("Validating the auth header")

			if err := authFN(authType, token); err == nil {
				log.Debug().Msg("Token is valid")
				return c.Next()
			}
		}

		log.Debug().Msg("Authentication failed")
		return fiber.ErrUnauthorized
	}
}

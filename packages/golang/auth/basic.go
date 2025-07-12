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

package auth

import (
	"encoding/base64"
	"fmt"
	"strings"
)

func HTTPBasic(username, password string) MiddlewareAuthFunction {
	return func(authType, authToken string) error {
		if authType == "Basic" {
			value, err := base64.StdEncoding.DecodeString(authToken)
			if err != nil {
				return fmt.Errorf("unable to decode base64 string: %w", err)
			}

			s := strings.Split(string(value), ":")
			if len(s) != 2 {
				return fmt.Errorf("incorrect format for basic token")
			}

			if username == s[0] && password == s[1] {
				// Valid
				return nil
			}

			return fmt.Errorf("invalid token")
		}
		return ErrInvalidAuthType
	}
}

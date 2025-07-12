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

func OneOf(fns ...MiddlewareAuthFunction) MiddlewareAuthFunction {
	count := len(fns)
	if count == 0 {
		panic("auth.OneOf must have at least one function")
	}

	return func(authType, authToken string) error {
		for k, fn := range fns {
			err := fn(authType, authToken)
			if err == nil {
				// Passed - don't continue
				return nil
			}
			if k == (count - 1) {
				// Last one has failed - fail all
				return err
			}
			// Failed, but not the last one - continue to next one
		}
		return nil
	}
}

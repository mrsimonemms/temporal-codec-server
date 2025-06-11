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
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

func TemporalJWKS(token string) error {
	return JWKS(token, TemporalIssuerURL)
}

func JWKS(token, jwksURL string) error {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(jwksURL)
	if err != nil {
		return fmt.Errorf("error retrieving jwks keys: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks response not ok: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading jwks body: %w", err)
	}

	k, err := keyfunc.NewJWKSetJSON(body)
	t, err := jwt.Parse(token, k.Keyfunc)
	if err != nil {
		return fmt.Errorf("error parsing jwt keys: %w", err)
	}

	if !t.Valid {
		return fmt.Errorf("token invalid")
	}

	return nil
}

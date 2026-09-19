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

package config

import "go.temporal.io/sdk/converter"

type Config struct {
	Auth       ConfigAuth       `json:"auth" envPrefix:"AUTH_"`
	Encryption ConfigEncryption `json:"encryption" envPrefix:"ENCRYPTION_"`
	Server     `json:"server" envPrefix:"SERVER_"`

	codecs []converter.PayloadCodec
}

type ConfigAuth struct {
	BasicUsername string `json:"basicUsername" env:"BASIC_USERNAME"`
	BasicPassword string `json:"basicPassword" env:"BASIC_PASSWORD"`
}

type ConfigEncryption struct {
	KeysPath   string   `json:"keysPath" env:"KEYS_PATH"`
	Namespaces []string `json:"namespaces,omitempty" env:"NAMESPACES"`
}

type Server struct {
	CORSAllowCreds bool     `json:"corsAllowCreds,omitempty" env:"CORS_ALLOW_CREDS"`
	CORSOrigins    []string `json:"corsOrigins,omitempty" env:"CORS_ORIGINS"`
	DisableAuth    bool     `json:"disableAuth,omitempty" env:"DISABLE_AUTH"`
	DisableCORS    bool     `json:"disableCors,omitempty" env:"DISABLE_CORS"`
	DisableSwagger bool     `json:"disableSwagger,omitempty" env:"DISABLE_SWAGGER"`
	Host           string   `json:"host,omitempty" env:"HOST"`
	Port           int      `json:"port" env:"PORT" validate:"gt=1"`
}

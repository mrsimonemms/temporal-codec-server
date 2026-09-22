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

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"sigs.k8s.io/yaml"
)

func (c *Config) GetCodecs() []converter.PayloadCodec {
	return c.codecs
}

func (c *Config) GetEncoders() map[string][]converter.PayloadCodec {
	encoders := map[string][]converter.PayloadCodec{
		client.DefaultNamespace: c.GetCodecs(),
	}

	for _, nsp := range c.Encryption.Namespaces {
		encoders[nsp] = c.GetCodecs()
	}

	return encoders
}

func (c *Config) loadAlgorithms() error {
	keys, err := aes.ReadKeyFile(c.Encryption.KeysPath)
	if err != nil {
		return fmt.Errorf("unable to get keys from file (%s): %w", c.Encryption.KeysPath, err)
	}
	c.codecs = append(c.codecs, aes.NewPayloadCodec(keys))

	return nil
}

func (c *Config) ToJSON() ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

func (c *Config) ToYAML() ([]byte, error) {
	return yaml.Marshal(c)
}

func (c *Config) Validate(ctx context.Context) error {
	validate := validator.New(
		validator.WithRequiredStructEnabled(),
		validator.WithTagNameFuncBlankOmit(),
	)

	return validate.StructCtx(ctx, c)
}

func Init() (*Config, error) {
	return &Config{
		Encryption: ConfigEncryption{
			KeysPath: "/path/to/keys.yaml",
		},
		Server: Server{
			CORSAllowCreds: true,
			CORSOrigins:    []string{"https://cloud.temporal.io"},
			Port:           3000,
		},
	}, nil
}

func Load(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("config path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error loading config file: %w", err)
	}

	var cfg Config
	expanded := os.Expand(string(data), os.Getenv)
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	if err := env.ParseWithOptions(&cfg, env.Options{
		SetDefaultsForZeroValuesOnly: true,
	}); err != nil {
		return nil, fmt.Errorf("error loading config envvars: %w", err)
	}

	// Load algorithms
	if err := cfg.loadAlgorithms(); err != nil {
		return nil, fmt.Errorf("error loading algorithms: %w", err)
	}

	return &cfg, nil
}

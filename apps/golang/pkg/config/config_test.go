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

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mrsimonemms/temporal-codec-server/apps/golang/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

// writeConfig writes a keys file and a config file referring to it, returning
// the config path.
func writeConfig(t *testing.T, body string) string {
	t.Helper()

	dir := t.TempDir()

	keysPath := filepath.Join(dir, "keys.yaml")
	keys := "- id: key0\n  key: passphrasewhichneedstobe32bytes!\n"
	require.NoError(t, os.WriteFile(keysPath, []byte(keys), 0o600))

	cfgPath := filepath.Join(dir, "config.yaml")
	cfg := "encryption:\n  keysPath: " + keysPath + "\n" + body
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfg), 0o600))

	return cfgPath
}

const (
	devNamespace  = "myapp-dev.acctid123"
	prodNamespace = "myapp-prod.acctid123"
)

func TestLoad_BuildsCodecForDefaultNamespace(t *testing.T) {
	cfg, err := config.Load(writeConfig(t, "server:\n  port: 3000\n"))
	require.NoError(t, err)

	require.Len(t, cfg.GetCodecs(), 1)

	encoders := cfg.GetEncoders()
	assert.Contains(t, encoders, client.DefaultNamespace)
	assert.Len(t, encoders[client.DefaultNamespace], 1)
}

// encryption.namespaces exists to let a single codec server serve more than
// one Temporal namespace, which is what the /{namespace}/encode and
// /{namespace}/decode routes are for. Every configured namespace needs a codec
// chain, otherwise its namespaced route can only ever 404.
func TestLoad_BuildsCodecForEveryConfiguredNamespace(t *testing.T) {
	body := "  namespaces:\n    - " + devNamespace + "\n    - " + prodNamespace + "\nserver:\n  port: 3000\n"

	cfg, err := config.Load(writeConfig(t, body))
	require.NoError(t, err)

	require.Equal(t, []string{devNamespace, prodNamespace}, cfg.Encryption.Namespaces,
		"the namespaces must be parsed from the config file")

	encoders := cfg.GetEncoders()
	for _, namespace := range cfg.Encryption.Namespaces {
		assert.Contains(t, encoders, namespace)
		assert.NotEmpty(t, encoders[namespace])
	}

	// Naming namespaces explicitly must not remove the default chain. The
	// router has no fallback of its own, so a missing default entry would
	// make a bare /encode or /decode 404.
	assert.Contains(t, encoders, client.DefaultNamespace)
	assert.NotEmpty(t, encoders[client.DefaultNamespace])
	assert.Len(t, encoders, 3)
}

// Namespace lists are operator-supplied, so the awkward values must resolve to
// something sane rather than to an extra or missing handler.
func TestLoad_NamespaceListEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		// namespaces is the YAML list body under encryption.namespaces.
		namespaces []string
		// wantKeys is every namespace expected to have a codec chain.
		wantKeys []string
	}{
		{
			name:       "duplicates collapse",
			namespaces: []string{devNamespace, devNamespace},
			wantKeys:   []string{client.DefaultNamespace, devNamespace},
		},
		{
			name:       "naming the default namespace is harmless",
			namespaces: []string{client.DefaultNamespace},
			wantKeys:   []string{client.DefaultNamespace},
		},
		{
			name:       "empty list leaves only the default",
			namespaces: nil,
			wantKeys:   []string{client.DefaultNamespace},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := "  namespaces:\n"
			for _, namespace := range test.namespaces {
				body += "    - " + namespace + "\n"
			}
			body += "server:\n  port: 3000\n"

			cfg, err := config.Load(writeConfig(t, body))
			require.NoError(t, err)

			encoders := cfg.GetEncoders()
			require.Len(t, encoders, len(test.wantKeys))
			for _, namespace := range test.wantKeys {
				assert.Contains(t, encoders, namespace)
				assert.NotEmpty(t, encoders[namespace])
			}
		})
	}
}

func TestLoad_Errors(t *testing.T) {
	t.Run("no path", func(t *testing.T) {
		_, err := config.Load("")

		assert.Error(t, err)
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := config.Load(filepath.Join(t.TempDir(), "nope.yaml"))

		assert.Error(t, err)
	})

	t.Run("missing keys file", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		cfg := "encryption:\n  keysPath: " + filepath.Join(dir, "nope.yaml") + "\nserver:\n  port: 3000\n"
		require.NoError(t, os.WriteFile(cfgPath, []byte(cfg), 0o600))

		_, err := config.Load(cfgPath)

		assert.Error(t, err)
	})
}

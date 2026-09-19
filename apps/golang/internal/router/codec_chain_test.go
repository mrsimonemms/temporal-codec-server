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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/converter"
)

const (
	codecOuter = "outer"
	codecInner = "inner"
)

// Each route must drive the configured chain in the direction the SDK
// defines: /encode applies codecs last to first, so the earlier codecs wrap
// the later ones, and /decode applies them first to last to unwind that.
//
// See converter.encodePayloads and converter.decodePayloads.
func TestCodecChain_Ordering(t *testing.T) {
	tests := []struct {
		name string
		// target is the route under test.
		target string
		// wantOrder is the sequence of codec invocations expected.
		wantOrder []string
		// wantTag is the metadata key the last codec to run writes, and the
		// codec expected to have written it.
		wantTag   string
		wantCodec string
	}{
		{
			name:      "encode",
			target:    pathEncode,
			wantOrder: []string{codecInner + ":encode", codecOuter + ":encode"},
			wantTag:   "encoded-by",
			wantCodec: codecOuter,
		},
		{
			name:      "decode",
			target:    pathDecode,
			wantOrder: []string{codecOuter + ":decode", codecInner + ":decode"},
			wantTag:   "decoded-by",
			wantCodec: codecInner,
		},
		{
			name:      "namespaced encode",
			target:    pathNamespacedEncode,
			wantOrder: []string{codecInner + ":encode", codecOuter + ":encode"},
			wantTag:   "encoded-by",
			wantCodec: codecOuter,
		},
		{
			name:      "namespaced decode",
			target:    pathNamespacedDecode,
			wantOrder: []string{codecOuter + ":decode", codecInner + ":decode"},
			wantTag:   "decoded-by",
			wantCodec: codecInner,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var order []string
			app := newTestApp(t, &Config{
				Encoders: map[string][]converter.PayloadCodec{
					defaultNamespace: {
						&recordingCodec{name: codecOuter, order: &order},
						&recordingCodec{name: codecInner, order: &order},
					},
				},
			})

			res := post(t, app, test.target, payloadsJSON(t, plainPayload(t, "hello")))
			require.Equal(t, http.StatusOK, res.status, res.body)

			assert.Equal(t, test.wantOrder, order)

			payloads := parsePayloads(t, res.body)
			require.Len(t, payloads, 1)
			assert.Equal(t, test.wantCodec, string(payloads[0].GetMetadata()[test.wantTag]))
		})
	}
}

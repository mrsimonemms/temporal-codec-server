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
	"sync"
	"testing"

	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/converter"
)

// The codec handlers are built once in New and shared by every request, so
// each one is used concurrently by the whole server rather than by a single
// request. This asserts that reuse is correct: run under -race, it covers the
// SDK handler, the fiber adaptor and the AES codec all being driven in
// parallel, and it proves that two namespaces sharing the handler map do not
// answer each other's requests.
//
// The Web UI issues many requests per Workflow Execution, so this is the
// normal operating mode, not an edge case.
func TestCodecHandlers_ReusedConcurrently(t *testing.T) {
	app := newTestApp(t, &Config{
		Encoders: map[string][]converter.PayloadCodec{
			defaultNamespace: {aes.NewPayloadCodec(testKeys)},
			otherNamespace:   {&recordingCodec{name: codecSecondary}},
		},
	})

	plaintext := payloadsJSON(t, plainPayload(t, "hello"))
	encoded := payloadsJSON(t, encodeForTest(t, plainPayload(t, "hello"))...)

	const workers = 24

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// The AES namespace must round trip correctly every time. A nonce
			// or cipher shared across goroutines would show up here.
			enc := post(t, app, pathEncode, plaintext)
			if !assert.Equal(t, http.StatusOK, enc.status, enc.body) {
				return
			}
			roundTripped := post(t, app, pathDecode, enc.body)
			if assert.Equal(t, http.StatusOK, roundTripped.status, roundTripped.body) {
				payloads := parsePayloads(t, roundTripped.body)
				if assert.Len(t, payloads, 1) {
					assert.JSONEq(t, `"hello"`, string(payloads[0].GetData()))
				}
			}

			dec := post(t, app, pathDecode, encoded)
			if assert.Equal(t, http.StatusOK, dec.status, dec.body) {
				payloads := parsePayloads(t, dec.body)
				if assert.Len(t, payloads, 1) {
					assert.JSONEq(t, `"hello"`, string(payloads[0].GetData()))
				}
			}

			// Interleaved requests for the other namespace must keep using the
			// other namespace's codec.
			other := post(t, app, "/"+otherNamespace+pathDecode, plaintext)
			if assert.Equal(t, http.StatusOK, other.status, other.body) {
				payloads := parsePayloads(t, other.body)
				if assert.Len(t, payloads, 1) {
					assert.Equal(t, codecSecondary, string(payloads[0].GetMetadata()["decoded-by"]))
				}
			}
		}()
	}
	wg.Wait()
}

// New builds one handler per configured namespace, and they must be distinct
// so that a request can never be routed to another namespace's codec chain.
func TestCodecHandlers_OnePerNamespace(t *testing.T) {
	cfg := &Config{
		Encoders: map[string][]converter.PayloadCodec{
			defaultNamespace: {&recordingCodec{name: codecPrimary}},
			otherNamespace:   {&recordingCodec{name: codecSecondary}},
		},
	}

	newTestApp(t, cfg)

	handlers := cfg.GetCodecHandlers()
	require.Len(t, handlers, 2)
	assert.Contains(t, handlers, defaultNamespace)
	assert.Contains(t, handlers, otherNamespace)
	assert.NotSame(t, handlers[defaultNamespace], handlers[otherNamespace])
}

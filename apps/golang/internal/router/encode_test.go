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

	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

// The Temporal Codec Server protocol requires /encode to accept a POST of a
// JSON envelope containing a "payloads" array, and to answer with the same
// envelope shape.
//
// https://docs.temporal.io/codec-server
// https://docs.temporal.io/production-deployment/data-encryption#api-contract-specifications
func TestEncode_SinglePayload(t *testing.T) {
	for _, target := range []string{pathEncode, pathNamespacedEncode} {
		t.Run(target, func(t *testing.T) {
			app := newAESApp(t)

			res := post(t, app, target, payloadsJSON(t, plainPayload(t, "hello")))

			require.Equal(t, http.StatusOK, res.status, res.body)
			assert.Equal(t, fiberJSONContentType, res.headers.Get("Content-Type"))

			payloads := parsePayloads(t, res.body)
			require.Len(t, payloads, 1)
			assert.Equal(t, aes.AESMimeType, encodingOf(payloads[0]))
			assert.Equal(t, "key0", string(payloads[0].GetMetadata()[aes.MetadataKeyID]))
			assert.NotEmpty(t, payloads[0].GetData())
		})
	}
}

// The codec server must be the inverse of itself: whatever /encode produces
// must survive a round trip through /decode.
func TestEncode_RoundTripsThroughDecode(t *testing.T) {
	app := newAESApp(t)

	encoded := post(t, app, pathEncode, payloadsJSON(t, plainPayload(t, "round trip")))
	require.Equal(t, http.StatusOK, encoded.status, encoded.body)

	decoded := post(t, app, pathDecode, encoded.body)
	require.Equal(t, http.StatusOK, decoded.status, decoded.body)

	payloads := parsePayloads(t, decoded.body)
	require.Len(t, payloads, 1)
	assert.Equal(t, converter.MetadataEncodingJSON, encodingOf(payloads[0]))
	assert.JSONEq(t, `"round trip"`, string(payloads[0].GetData()))
}

// Payload order and cardinality are part of the contract: the SDK's remote
// codec rejects a response whose payload count differs from the request.
//
// See remotePayloadCodec.encodeOrDecode in go.temporal.io/sdk/converter.
func TestEncode_MultiplePayloads(t *testing.T) {
	app := newAESApp(t)

	inputs := []*commonpb.Payload{
		plainPayload(t, "first"),
		plainPayload(t, "second"),
		plainPayload(t, "third"),
	}

	res := post(t, app, pathEncode, payloadsJSON(t, inputs...))
	require.Equal(t, http.StatusOK, res.status, res.body)

	encoded := parsePayloads(t, res.body)
	require.Len(t, encoded, len(inputs))
	for _, p := range encoded {
		assert.Equal(t, aes.AESMimeType, encodingOf(p))
	}

	// Decoding must give the originals back, in the original order.
	decoded := post(t, app, pathDecode, res.body)
	require.Equal(t, http.StatusOK, decoded.status, decoded.body)

	out := parsePayloads(t, decoded.body)
	require.Len(t, out, len(inputs))
	for i, want := range []string{`"first"`, `"second"`, `"third"`} {
		assert.JSONEq(t, want, string(out[i].GetData()))
	}
}

// An empty payload list is legal and must not be treated as an error.
func TestEncode_EmptyPayloadList(t *testing.T) {
	app := newAESApp(t)

	res := post(t, app, pathEncode, `{"payloads":[]}`)

	require.Equal(t, http.StatusOK, res.status, res.body)
	assert.Equal(t, fiberJSONContentType, res.headers.Get("Content-Type"))
	assert.Empty(t, parsePayloads(t, res.body))

	// protojson omits empty repeated fields, so the envelope comes back as an
	// empty object rather than {"payloads":[]}. This matches the upstream
	// converter.NewPayloadCodecHTTPHandler behaviour exactly.
	assert.Empty(t, rawEnvelope(t, res.body))
}

// A body with no "payloads" key at all behaves as an empty list.
func TestEncode_MissingPayloadsKey(t *testing.T) {
	app := newAESApp(t)

	res := post(t, app, pathEncode, `{}`)

	require.Equal(t, http.StatusOK, res.status, res.body)
	assert.Empty(t, parsePayloads(t, res.body))
}

// A codec that errors must surface as a 4xx, not a 500 or a silent success.
func TestEncode_CodecFailure(t *testing.T) {
	app := newTestApp(t, &Config{
		Encoders: map[string][]converter.PayloadCodec{
			defaultNamespace: {failingCodec{}},
		},
	})

	res := post(t, app, pathEncode, payloadsJSON(t, plainPayload(t, "hello")))

	assert.Equal(t, http.StatusBadRequest, res.status)
	assert.Contains(t, res.body, "encode exploded")
}

// Malformed requests must be rejected with 400 rather than panicking or
// returning a misleading 200.
func TestEncode_MalformedRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: notJSON, body: `this is not json`},
		{name: "empty body", body: ``},
		{name: "payloads is a string", body: `{"payloads":"nope"}`},
		{name: "payloads is an object", body: `{"payloads":{}}`},
		{name: "payload is a string", body: `{"payloads":["nope"]}`},
		{name: "data is not base64", body: `{"payloads":[{"data":"!!!not-base64!!!"}]}`},
		{name: "unknown field", body: `{"payloads":[],"nope":true}`},
		{name: "truncated json", body: `{"payloads":[{"data":`},
	}

	app := newAESApp(t)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := post(t, app, pathEncode, test.body)

			assert.Equal(t, http.StatusBadRequest, res.status, res.body)
		})
	}
}

// Payloads with no metadata are valid; the AES codec encrypts them like any
// other payload.
func TestEncode_PayloadWithoutMetadata(t *testing.T) {
	app := newAESApp(t)

	res := post(t, app, pathEncode, `{"payloads":[{"data":"ImEi"}]}`)

	require.Equal(t, http.StatusOK, res.status, res.body)
	payloads := parsePayloads(t, res.body)
	require.Len(t, payloads, 1)
	assert.Equal(t, aes.AESMimeType, encodingOf(payloads[0]))
}

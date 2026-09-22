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

// encodeForTest runs payloads through the AES codec directly so that decode
// tests have realistic encoded input without depending on /encode.
func encodeForTest(t *testing.T, payloads ...*commonpb.Payload) []*commonpb.Payload {
	t.Helper()

	encoded, err := aes.NewPayloadCodec(testKeys).Encode(payloads)
	require.NoError(t, err)

	return encoded
}

func TestDecode_SinglePayload(t *testing.T) {
	for _, target := range []string{pathDecode, pathNamespacedDecode} {
		t.Run(target, func(t *testing.T) {
			app := newAESApp(t)
			encoded := encodeForTest(t, plainPayload(t, "hello"))

			res := post(t, app, target, payloadsJSON(t, encoded...))

			require.Equal(t, http.StatusOK, res.status, res.body)
			assert.Equal(t, fiberJSONContentType, res.headers.Get("Content-Type"))

			payloads := parsePayloads(t, res.body)
			require.Len(t, payloads, 1)
			assert.Equal(t, converter.MetadataEncodingJSON, encodingOf(payloads[0]))
			assert.JSONEq(t, `"hello"`, string(payloads[0].GetData()))
		})
	}
}

func TestDecode_MultiplePayloads(t *testing.T) {
	app := newAESApp(t)
	encoded := encodeForTest(t,
		plainPayload(t, "first"),
		plainPayload(t, "second"),
		plainPayload(t, "third"),
	)

	res := post(t, app, pathDecode, payloadsJSON(t, encoded...))
	require.Equal(t, http.StatusOK, res.status, res.body)

	payloads := parsePayloads(t, res.body)
	require.Len(t, payloads, 3)
	for i, want := range []string{`"first"`, `"second"`, `"third"`} {
		assert.JSONEq(t, want, string(payloads[i].GetData()))
	}
}

// A payload that this codec did not produce must pass through untouched. The
// SDK requires codecs not to decode payloads they did not encode.
func TestDecode_LeavesForeignPayloadsAlone(t *testing.T) {
	app := newAESApp(t)
	plain := plainPayload(t, "never encrypted")

	res := post(t, app, pathDecode, payloadsJSON(t, plain))
	require.Equal(t, http.StatusOK, res.status, res.body)

	payloads := parsePayloads(t, res.body)
	require.Len(t, payloads, 1)
	assert.Equal(t, converter.MetadataEncodingJSON, encodingOf(payloads[0]))
	assert.JSONEq(t, `"never encrypted"`, string(payloads[0].GetData()))
}

func TestDecode_MixedEncodedAndPlainPayloads(t *testing.T) {
	app := newAESApp(t)

	encoded := encodeForTest(t, plainPayload(t, "secret"))
	body := payloadsJSON(t, encoded[0], plainPayload(t, "public"))

	res := post(t, app, pathDecode, body)
	require.Equal(t, http.StatusOK, res.status, res.body)

	payloads := parsePayloads(t, res.body)
	require.Len(t, payloads, 2)
	assert.JSONEq(t, `"secret"`, string(payloads[0].GetData()))
	assert.JSONEq(t, `"public"`, string(payloads[1].GetData()))
}

func TestDecode_EmptyPayloadList(t *testing.T) {
	app := newAESApp(t)

	res := post(t, app, pathDecode, `{"payloads":[]}`)

	require.Equal(t, http.StatusOK, res.status, res.body)
	assert.Equal(t, fiberJSONContentType, res.headers.Get("Content-Type"))
	assert.Empty(t, parsePayloads(t, res.body))
}

func TestDecode_CodecFailure(t *testing.T) {
	app := newTestApp(t, &Config{
		Encoders: map[string][]converter.PayloadCodec{
			defaultNamespace: {failingCodec{}},
		},
	})

	res := post(t, app, pathDecode, payloadsJSON(t, plainPayload(t, "hello")))

	assert.Equal(t, http.StatusBadRequest, res.status)
	assert.Contains(t, res.body, "decode exploded")
}

// An AES payload referencing a key the server does not hold is a codec
// failure, not a malformed request, but must still be a 4xx.
func TestDecode_UnknownEncryptionKey(t *testing.T) {
	app := newAESApp(t)

	foreign := &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte(aes.AESMimeType),
			aes.MetadataKeyID:          []byte("key-we-do-not-have"),
		},
		Data: []byte("whatever"),
	}

	res := post(t, app, pathDecode, payloadsJSON(t, foreign))

	assert.Equal(t, http.StatusBadRequest, res.status)
	assert.Contains(t, res.body, "unknown encryption key")
}

// A payload that claims our encoding but carries too few bytes to hold a GCM
// nonce must be a decode failure, not a 200 with the payloads missing.
//
// Regression: the AES codec used to return (nil, nil) here, which the SDK
// handler marshalled as an empty envelope. The caller got 200 {} and its
// payloads silently vanished; the SDK's remote codec then failed with
// "received 0 payloads from remote codec, expected 2".
func TestDecode_ShortCiphertextIsAnError(t *testing.T) {
	app := newAESApp(t)

	malformed := &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte(aes.AESMimeType),
			aes.MetadataKeyID:          []byte("key0"),
		},
		// Eight bytes, under the 12-byte AES-GCM nonce.
		Data: []byte("tooshort"),
	}
	encoded := encodeForTest(t, plainPayload(t, "sibling"))

	res := post(t, app, pathDecode, payloadsJSON(t, malformed, encoded[0]))

	require.Equal(t, http.StatusBadRequest, res.status, res.body)
	assert.Contains(t, res.body, "shorter than")
	assert.NotEqual(t, "{}\n", res.body, "the payloads must not be dropped silently")
}

func TestDecode_MalformedRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: notJSON, body: `this is not json`},
		{name: "empty body", body: ``},
		{name: "payloads is a string", body: `{"payloads":"nope"}`},
		{name: "payload is a number", body: `{"payloads":[1]}`},
		{name: "metadata is not a map", body: `{"payloads":[{"metadata":"nope"}]}`},
		{name: "data is not base64", body: `{"payloads":[{"data":"!!!not-base64!!!"}]}`},
		{name: "json array at top level", body: `[]`},
	}

	app := newAESApp(t)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := post(t, app, pathDecode, test.body)

			assert.Equal(t, http.StatusBadRequest, res.status, res.body)
		})
	}
}

// ############################################################ //
// /decode?preserveStorageRefs=true                              //
//                                                               //
// Current Temporal behaviour: when the flag is set, External     //
// Storage references must be returned untouched while ordinary   //
// payloads are still decoded. The Web UI relies on this to show  //
// reference metadata before the user asks for the full payload.  //
//                                                                //
// https://docs.temporal.io/codec-server#external-storage         //
// go.temporal.io/sdk/converter.(*payloadHTTPHandler).decode       //
// ############################################################ //

// A storage reference must survive /decode?preserveStorageRefs=true as a
// storage reference, while an ordinary payload in the same batch is decoded.
func TestDecode_PreserveStorageRefs(t *testing.T) {
	targets := []string{
		pathDecodePreserveRefs,
		// The SDK compares the value case-insensitively.
		"/decode?preserveStorageRefs=TRUE",
		"/default/decode?preserveStorageRefs=true",
	}

	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			app := newAESApp(t)

			encoded := encodeForTest(t, plainPayload(t, "ordinary"))
			body := payloadsJSON(t, storageRefPayload("s3"), encoded[0])

			res := post(t, app, target, body)
			require.Equal(t, http.StatusOK, res.status, res.body)
			assert.Equal(t, fiberJSONContentType, res.headers.Get("Content-Type"))

			payloads := parsePayloads(t, res.body)
			require.Len(t, payloads, 2)

			// The reference is still a reference.
			assert.Equal(t, converter.MetadataEncodingProtoJSON, encodingOf(payloads[0]))
			assert.Equal(t,
				externalStorageReferenceMessageType,
				string(payloads[0].GetMetadata()[converter.MetadataMessageType]),
			)
			assert.JSONEq(t, `{"driverName":"s3"}`, string(payloads[0].GetData()))

			// The ordinary payload is decoded.
			assert.JSONEq(t, `"ordinary"`, string(payloads[1].GetData()))
		})
	}
}

// This server configures no External Storage, so it has no way to resolve a
// storage reference and correctly leaves references alone on every /decode,
// with or without the flag. The AES codec ignores them because their encoding
// is not its own.
//
// This is the whole observable contract today. It is NOT proof that
// preserveStorageRefs is handled: codecConverter never reads the query string.
// The distinction only becomes visible if reference-resolving decode is ever
// added, at which point the flag must start suppressing that resolution.
func TestDecode_StorageRefsPassThroughWithoutExternalStorage(t *testing.T) {
	targets := []string{
		pathDecode,
		"/decode?preserveStorageRefs=false",
		pathDecodePreserveRefs,
	}

	ref := storageRefPayload("s3")

	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			app := newAESApp(t)

			res := post(t, app, target, payloadsJSON(t, ref))
			require.Equal(t, http.StatusOK, res.status, res.body)

			payloads := parsePayloads(t, res.body)
			require.Len(t, payloads, 1)
			assert.Equal(t, converter.MetadataEncodingProtoJSON, encodingOf(payloads[0]))
			assert.Equal(t,
				externalStorageReferenceMessageType,
				string(payloads[0].GetMetadata()[converter.MetadataMessageType]),
			)
			assert.JSONEq(t, `{"driverName":"s3"}`, string(payloads[0].GetData()))
		})
	}
}

// Query parameters the server does not act on must be ignored rather than
// rejected or allowed to break the decode.
func TestDecode_IrrelevantQueryParameters(t *testing.T) {
	targets := []string{
		"/decode?preserveStorageRefs=banana",
		"/decode?preserveStorageRefs",
		"/decode?somethingElse=1",
		"/decode?preserveStorageRefs=true&somethingElse=1",
	}

	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			app := newAESApp(t)
			encoded := encodeForTest(t, plainPayload(t, "hello"))

			res := post(t, app, target, payloadsJSON(t, encoded...))

			require.Equal(t, http.StatusOK, res.status, res.body)
			payloads := parsePayloads(t, res.body)
			require.Len(t, payloads, 1)
			assert.JSONEq(t, `"hello"`, string(payloads[0].GetData()))
		})
	}
}

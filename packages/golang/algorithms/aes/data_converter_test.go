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

package aes_test

import (
	"testing"

	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

// gcmNonceSize is the nonce length AES-GCM uses. Ciphertext shorter than this
// cannot possibly carry a nonce, let alone an auth tag.
const gcmNonceSize = 12

var testKeys = aes.Keys{
	{ID: "key0", Key: "passphrasewhichneedstobe32bytes!"},
	{ID: "key1", Key: "anoldpassphraseinourhistory!!!!!"},
}

func newCodec() converter.PayloadCodec {
	return aes.NewPayloadCodec(testKeys)
}

// plainPayload builds an unencoded payload, as a Temporal client would produce
// before any codec runs.
func plainPayload(t *testing.T, value any) *commonpb.Payload {
	t.Helper()

	p, err := converter.GetDefaultDataConverter().ToPayload(value)
	require.NoError(t, err)

	return p
}

// encrypted runs value through Encode so tests have genuine ciphertext.
func encrypted(t *testing.T, value any) *commonpb.Payload {
	t.Helper()

	out, err := newCodec().Encode([]*commonpb.Payload{plainPayload(t, value)})
	require.NoError(t, err)
	require.Len(t, out, 1)

	return out[0]
}

// malformedEncrypted builds a payload that claims to be AES encrypted, names a
// key the codec holds, but carries too few bytes to contain a nonce.
func malformedEncrypted(data []byte) *commonpb.Payload {
	return &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte(aes.AESMimeType),
			aes.MetadataKeyID:          []byte("key0"),
		},
		Data: data,
	}
}

// Ciphertext shorter than the GCM nonce must be reported as a decode failure.
//
// Returning (nil, nil) here is the regression this test exists for: the
// caller cannot distinguish it from a successful decode of zero payloads, so
// the payloads disappear instead of the error surfacing.
func TestDecode_ShortCiphertextReturnsError(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "nil data", data: nil},
		{name: "empty data", data: []byte{}},
		{name: "one byte", data: []byte{0x01}},
		{name: "one byte short of the nonce", data: make([]byte, gcmNonceSize-1)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Less(t, len(test.data), gcmNonceSize, "the fixture must be shorter than the nonce")

			out, err := newCodec().Decode([]*commonpb.Payload{malformedEncrypted(test.data)})

			require.Error(t, err, "a malformed ciphertext must be an error, not a silent success")
			assert.Nil(t, out, "no payload may be returned alongside a decode error")
			assert.ErrorContains(t, err, "shorter than")
		})
	}
}

// The specific shape of the bug: Decode must never report success while
// dropping the payloads it was asked to decode.
func TestDecode_NeverReturnsNilResultWithNilError(t *testing.T) {
	malformed := []*commonpb.Payload{
		malformedEncrypted(nil),
		malformedEncrypted([]byte("short")),
		malformedEncrypted(make([]byte, gcmNonceSize-1)),
	}

	for _, payload := range malformed {
		out, err := newCodec().Decode([]*commonpb.Payload{payload})

		if err == nil {
			require.NotNil(t, out, "Decode returned (nil, nil): success with no output")
		}
		assert.Error(t, err)
	}
}

// A single bad payload in a batch must not collapse the whole response. The
// SDK's remote codec rejects a reply whose payload count differs from the
// request, so a short result is worse than an error.
func TestDecode_MalformedPayloadInBatchDoesNotCollapseOutput(t *testing.T) {
	batch := []*commonpb.Payload{
		encrypted(t, "first"),
		malformedEncrypted([]byte("short")),
		encrypted(t, "third"),
	}

	out, err := newCodec().Decode(batch)

	require.Error(t, err, "the batch must fail rather than return fewer payloads than it was given")
	assert.Nil(t, out)

	// The failure must not be reported as a partial success either.
	if err == nil {
		assert.Len(t, out, len(batch))
	}
}

// Exactly nonce-length data passes the length check but carries no auth tag,
// so GCM must reject it. This is the boundary either side of the fix.
func TestDecode_CiphertextOfExactlyNonceLength(t *testing.T) {
	out, err := newCodec().Decode([]*commonpb.Payload{
		malformedEncrypted(make([]byte, gcmNonceSize)),
	})

	require.Error(t, err)
	assert.Nil(t, out)
}

// Tampered ciphertext of a legitimate length must fail authentication.
func TestDecode_TamperedCiphertextFailsAuthentication(t *testing.T) {
	payload := encrypted(t, "hello")
	payload.Data[len(payload.Data)-1] ^= 0xFF

	out, err := newCodec().Decode([]*commonpb.Payload{payload})

	require.Error(t, err)
	assert.Nil(t, out)
}

// ######################################################## //
// The fix must not disturb any of the following behaviour. //
// ######################################################## //

func TestEncodeDecode_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{name: "string", value: "hello"},
		{name: "empty string", value: ""},
		{name: "number", value: 42},
		{name: "bool", value: true},
		{name: "nil", value: nil},
		{name: "struct", value: map[string]any{"a": 1.0, "b": "two"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			codec := newCodec()
			original := plainPayload(t, test.value)

			encodedPayloads, err := codec.Encode([]*commonpb.Payload{original})
			require.NoError(t, err)
			require.Len(t, encodedPayloads, 1)

			// The encryption format is unchanged: our encoding hint, the
			// active key id, and a body at least as long as the nonce.
			assert.Equal(t, aes.AESMimeType,
				string(encodedPayloads[0].Metadata[converter.MetadataEncoding]))
			assert.Equal(t, "key0", string(encodedPayloads[0].Metadata[aes.MetadataKeyID]))
			assert.GreaterOrEqual(t, len(encodedPayloads[0].Data), gcmNonceSize)

			decoded, err := codec.Decode(encodedPayloads)
			require.NoError(t, err)
			require.Len(t, decoded, 1)

			assert.Equal(t, original.Metadata, decoded[0].Metadata)
			assert.Equal(t, original.Data, decoded[0].Data)
		})
	}
}

func TestEncodeDecode_RoundTripMultiplePayloads(t *testing.T) {
	codec := newCodec()
	originals := []*commonpb.Payload{
		plainPayload(t, "first"),
		plainPayload(t, "second"),
		plainPayload(t, "third"),
	}

	encodedPayloads, err := codec.Encode(originals)
	require.NoError(t, err)
	require.Len(t, encodedPayloads, len(originals))

	decoded, err := codec.Decode(encodedPayloads)
	require.NoError(t, err)
	require.Len(t, decoded, len(originals))

	for i, original := range originals {
		assert.Equal(t, original.Data, decoded[i].Data)
	}
}

// Empty batches are legal and must not error.
func TestEncodeDecode_EmptyBatch(t *testing.T) {
	codec := newCodec()

	encodedPayloads, err := codec.Encode([]*commonpb.Payload{})
	require.NoError(t, err)
	assert.Empty(t, encodedPayloads)

	decoded, err := codec.Decode([]*commonpb.Payload{})
	require.NoError(t, err)
	assert.Empty(t, decoded)
}

// Payloads this codec did not encode must pass through untouched, including
// ones whose data is shorter than a nonce. Only payloads claiming our encoding
// go anywhere near the cipher.
func TestDecode_ForeignPayloadsPassThrough(t *testing.T) {
	tests := []struct {
		name    string
		payload *commonpb.Payload
	}{
		{name: "json/plain", payload: plainPayload(t, "never encrypted")},
		{
			name: "short data, foreign encoding",
			payload: &commonpb.Payload{
				Metadata: map[string][]byte{
					converter.MetadataEncoding: []byte("binary/plain"),
				},
				Data: []byte("x"),
			},
		},
		{name: "no metadata at all", payload: &commonpb.Payload{Data: []byte("x")}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out, err := newCodec().Decode([]*commonpb.Payload{test.payload})

			require.NoError(t, err)
			require.Len(t, out, 1)
			assert.Equal(t, test.payload.Data, out[0].Data)
		})
	}
}

// A payload encrypted with a retired key must still decode, which is the
// point of the key list.
func TestDecode_UsesTheKeyNamedInTheMetadata(t *testing.T) {
	// Encrypt with key1 by presenting it as the active key.
	rotated := aes.NewPayloadCodec(aes.Keys{testKeys[1]})

	encodedPayloads, err := rotated.Encode([]*commonpb.Payload{plainPayload(t, "old secret")})
	require.NoError(t, err)
	require.Equal(t, "key1", string(encodedPayloads[0].Metadata[aes.MetadataKeyID]))

	// The full key set, where key1 is no longer first, must still decode it.
	decoded, err := newCodec().Decode(encodedPayloads)
	require.NoError(t, err)
	require.Len(t, decoded, 1)
	assert.JSONEq(t, `"old secret"`, string(decoded[0].Data))
}

func TestDecode_KeyMetadataErrors(t *testing.T) {
	tests := []struct {
		name    string
		keyID   []byte
		wantErr string
	}{
		{name: "no key id", keyID: nil, wantErr: "no key id provided"},
		{name: "empty key id", keyID: []byte(""), wantErr: "no key id provided"},
		{name: "unknown key id", keyID: []byte("key99"), wantErr: "unknown encryption key"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := &commonpb.Payload{
				Metadata: map[string][]byte{
					converter.MetadataEncoding: []byte(aes.AESMimeType),
					aes.MetadataKeyID:          test.keyID,
				},
				Data: make([]byte, 64),
			}

			out, err := newCodec().Decode([]*commonpb.Payload{payload})

			require.Error(t, err)
			assert.Nil(t, out)
			assert.ErrorContains(t, err, test.wantErr)
		})
	}
}

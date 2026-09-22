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
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/converter"
)

const (
	basicUsername = "codec"
	basicPassword = "s3cret"
)

type authApp struct {
	app   *fiber.App
	codec *recordingCodec
}

func newAuthApp(t *testing.T) *authApp {
	t.Helper()

	codec := &recordingCodec{name: codecPrimary}
	app := newTestApp(t, &Config{
		EnableAuth:    true,
		BasicUsername: basicUsername,
		BasicPassword: basicPassword,
		Encoders: map[string][]converter.PayloadCodec{
			defaultNamespace: {codec},
		},
	})

	return &authApp{app: app, codec: codec}
}

func basicAuth(username, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}

// /decode returns plaintext, so it must be behind authentication when
// authentication is enabled. A query string must not change that.
func TestAuth_DecodeRequiresCredentials(t *testing.T) {
	targets := []string{
		pathDecode,
		pathNamespacedDecode,
		pathDecodePreserveRefs,
	}

	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			under := newAuthApp(t)

			res := post(t, under.app, target, payloadsJSON(t, plainPayload(t, "hello")))

			assert.Equal(t, http.StatusUnauthorized, res.status, res.body)
			assert.Zero(t, under.codec.decodeCallCount(),
				"the codec must not run for an unauthenticated request")
		})
	}
}

// The negative cases above are only meaningful if a correct credential is
// actually accepted. auth.OneOf tries the Temporal Cloud JWKS validator
// first, which rejects a non-Bearer credential type without any network
// call, then falls through to HTTP Basic.
func TestAuth_DecodeAcceptsValidCredentials(t *testing.T) {
	under := newAuthApp(t)

	res := post(t, under.app, pathDecode, payloadsJSON(t, plainPayload(t, "hello")),
		map[string]string{fiber.HeaderAuthorization: basicAuth(basicUsername, basicPassword)})

	require.Equal(t, http.StatusOK, res.status, res.body)
	assert.Equal(t, 1, under.codec.decodeCallCount())

	payloads := parsePayloads(t, res.body)
	require.Len(t, payloads, 1)
	assert.Equal(t, codecPrimary, string(payloads[0].GetMetadata()["decoded-by"]))
}

func TestAuth_DecodeRejectsBadCredentials(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{name: "wrong password", header: basicAuth(basicUsername, "nope")},
		{name: "wrong username", header: basicAuth("nobody", basicPassword)},
		{name: "not base64", header: "Basic !!!not-base64!!!"},
		{name: "no credential type", header: "justatoken"},
		{name: "unknown credential type", header: "Weird abc123"},
		{name: "type with no token", header: "Basic "},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			under := newAuthApp(t)

			res := post(t, under.app, pathDecode, payloadsJSON(t, plainPayload(t, "hello")),
				map[string]string{fiber.HeaderAuthorization: test.header})

			assert.Equal(t, http.StatusUnauthorized, res.status, res.body)
			assert.Zero(t, under.codec.decodeCallCount())
		})
	}
}

// This server deliberately leaves /encode unauthenticated: it only turns
// caller-supplied plaintext into ciphertext, so it discloses nothing the
// caller did not already have.
func TestAuth_EncodeIsUnauthenticated(t *testing.T) {
	targets := []string{
		pathEncode,
		pathNamespacedEncode,
		// The exemption matches on the routed path, so a query string must
		// not accidentally re-enable authentication.
		"/encode?preserveStorageRefs=false",
		"/encode?anything=1&else=2",
	}

	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			under := newAuthApp(t)

			res := post(t, under.app, target, payloadsJSON(t, plainPayload(t, "hello")))

			require.Equal(t, http.StatusOK, res.status, res.body)
			assert.Equal(t, 1, under.codec.encodeCallCount(),
				"the request must reach the codec, not just return 200")
		})
	}
}

// The exemption is a suffix match on the path, so it must not be widened by a
// namespace whose name contains or equals "encode". A namespace called
// "encode" makes the decode route /encode/decode, which must still require
// credentials.
func TestAuth_ExemptionIsNotWidenedByNamespaceName(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		path       string
		wantStatus int
	}{
		{
			name:       "namespace named encode, decode route",
			namespace:  "encode",
			path:       "/encode/decode",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "namespace ending in encode, decode route",
			namespace:  "myencode",
			path:       "/myencode/decode",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "namespace named decode, encode route stays exempt",
			namespace:  "decode",
			path:       "/decode/encode",
			wantStatus: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t, &Config{
				EnableAuth:    true,
				BasicUsername: basicUsername,
				BasicPassword: basicPassword,
				Encoders: map[string][]converter.PayloadCodec{
					test.namespace: {&recordingCodec{name: codecPrimary}},
				},
			})

			res := post(t, app, test.path, payloadsJSON(t, plainPayload(t, "hello")))

			assert.Equal(t, test.wantStatus, res.status, res.body)
		})
	}
}

// With authentication disabled every route is open.
func TestAuth_Disabled(t *testing.T) {
	app := newTestApp(t, &Config{
		EnableAuth: false,
		Encoders: map[string][]converter.PayloadCodec{
			defaultNamespace: {aes.NewPayloadCodec(testKeys)},
		},
	})

	targets := []string{pathEncode, pathDecode, pathNamespacedEncode, pathNamespacedDecode}

	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			res := post(t, app, target, `{"payloads":[]}`)

			assert.Equal(t, http.StatusOK, res.status, res.body)
		})
	}
}

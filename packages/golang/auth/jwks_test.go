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

package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mrsimonemms/temporal-codec-server/packages/golang/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testKeyID = "test-key"

	// testSubject is the claim an identity provider would put in the token
	// the Web UI forwards.
	testSubject = "someone@example.com"

	claimSubject = "sub"
	claimExpiry  = "exp"
)

// signingKey is a throwaway RSA key used to build a JWK Set and to sign the
// tokens validated against it.
type signingKey struct {
	private *rsa.PrivateKey
	keyID   string
}

func newSigningKey(t *testing.T) signingKey {
	t.Helper()

	// 2048 is the smallest size crypto/rsa will sign RS256 with, and keeping
	// it small keeps the test fast.
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return signingKey{private: private, keyID: testKeyID}
}

// jwksJSON renders the public half of the key as a JWK Set document, the same
// shape an OIDC provider's jwks.json serves.
func (k signingKey) jwksJSON(t *testing.T) string {
	t.Helper()

	public := k.private.PublicKey

	exponent := make([]byte, 4)
	binary.BigEndian.PutUint32(exponent, uint32(public.E)) //nolint:gosec // E is small
	exponent = trimLeadingZeros(exponent)

	jwks := map[string]any{
		"keys": []map[string]string{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": k.keyID,
				"n":   base64.RawURLEncoding.EncodeToString(public.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(exponent),
			},
		},
	}

	bs, err := json.Marshal(jwks)
	require.NoError(t, err)

	return string(bs)
}

func trimLeadingZeros(b []byte) []byte {
	for len(b) > 1 && b[0] == 0 {
		b = b[1:]
	}

	return b
}

// sign issues an RS256 token carrying the key id, so the key set can resolve
// the signing key.
func (k signingKey) sign(t *testing.T, claims jwt.Claims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = k.keyID

	signed, err := token.SignedString(k.private)
	require.NoError(t, err)

	return signed
}

// signSubject issues a token for the standard test subject, which is what
// every test needs unless it is exercising a specific claim.
func (k signingKey) signSubject(t *testing.T) string {
	t.Helper()

	return k.sign(t, jwt.MapClaims{claimSubject: testSubject})
}

// jwksServer serves body at a unique URL. The production code caches JWKS
// responses by URL, so every test needs its own server to avoid reading
// another test's cached document.
func jwksServer(t *testing.T, status int, body string) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)

	return server.URL
}

// A JWKS endpoint that answers 200 with something that is not a JWK Set must
// be reported as an error.
//
// The regression this guards: the parse error used to be discarded and the
// nil key set dereferenced, which panics. In the Codec Server that panic is
// caught by the recovery middleware and served as a 500, so a broken identity
// provider looked like a broken Codec Server instead of an auth failure.
func TestJWKS_MalformedKeySetReturnsError(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "not json", body: `this is not json`},
		{name: "empty body", body: ``},
		{name: "json array", body: `[]`},
		{name: "json string", body: `"nope"`},
		{name: "keys is not an array", body: `{"keys":"nope"}`},
		{name: "html error page", body: `<html><body>502 Bad Gateway</body></html>`},
		{name: "truncated json", body: `{"keys":[{"kty":"RSA"`},
	}

	key := newSigningKey(t)
	token := key.signSubject(t)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			url := jwksServer(t, http.StatusOK, test.body)

			// The call must return, not panic.
			var err error
			require.NotPanics(t, func() {
				err = auth.JWKS(token, url)
			})

			assert.Error(t, err, "a malformed JWK Set must be an authentication error")
		})
	}
}

// A key set that parses but contains no usable key must also fail cleanly:
// the token cannot be verified against it.
func TestJWKS_EmptyKeySetReturnsError(t *testing.T) {
	key := newSigningKey(t)
	token := key.signSubject(t)

	url := jwksServer(t, http.StatusOK, `{"keys":[]}`)

	var err error
	require.NotPanics(t, func() {
		err = auth.JWKS(token, url)
	})

	assert.Error(t, err)
}

// The negative cases above only mean something if a genuine JWKS and token
// pair still validates.
func TestJWKS_ValidKeySetAndTokenSucceeds(t *testing.T) {
	key := newSigningKey(t)
	token := key.signSubject(t)

	url := jwksServer(t, http.StatusOK, key.jwksJSON(t))

	assert.NoError(t, auth.JWKS(token, url))
}

func TestJWKS_ValidKeySetRejectsBadTokens(t *testing.T) {
	key := newSigningKey(t)
	other := newSigningKey(t)

	tests := []struct {
		name  string
		token string
	}{
		{name: "not a jwt", token: "garbage"},
		{name: "empty token", token: ""},
		{name: "signed by another key", token: other.signSubject(t)},
		{
			name:  "expired",
			token: key.sign(t, jwt.MapClaims{claimSubject: testSubject, claimExpiry: 1}),
		},
	}

	url := jwksServer(t, http.StatusOK, key.jwksJSON(t))

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var err error
			require.NotPanics(t, func() {
				err = auth.JWKS(test.token, url)
			})

			assert.Error(t, err)
		})
	}
}

// A non-200 from the JWKS endpoint is a fetch failure, not a parse failure,
// and must never reach the key set handling.
func TestJWKS_FetchFailures(t *testing.T) {
	key := newSigningKey(t)
	token := key.signSubject(t)

	t.Run("non-200 response", func(t *testing.T) {
		url := jwksServer(t, http.StatusInternalServerError, key.jwksJSON(t))

		var err error
		require.NotPanics(t, func() {
			err = auth.JWKS(token, url)
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "jwks")
	})

	t.Run("unreachable endpoint", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		url := server.URL
		server.Close()

		var err error
		require.NotPanics(t, func() {
			err = auth.JWKS(token, url)
		})

		assert.Error(t, err)
	})
}

// TemporalJWKS must reject a credential type it does not handle before doing
// any network work. The Codec Server relies on this: auth.OneOf tries this
// validator first, and HTTP Basic would be unusable if a Basic credential
// triggered a JWKS fetch.
func TestTemporalJWKS_RejectsOtherCredentialTypes(t *testing.T) {
	tests := []string{"Basic", "basic", "bearer", "Weird", ""}

	for _, authType := range tests {
		t.Run(authType, func(t *testing.T) {
			err := auth.TemporalJWKS(authType, "any-token")

			assert.ErrorIs(t, err, auth.ErrInvalidAuthType)
		})
	}
}

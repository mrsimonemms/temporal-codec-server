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
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrsimonemms/temporal-codec-server/apps/golang/auth"
	pkgauth "github.com/mrsimonemms/temporal-codec-server/packages/golang/auth"
)

const (
	testKeyID = "test-key"
	bearer    = "Bearer"
)

// countingTransport records every outbound HTTP request the validator makes so
// that tests can assert an untrusted token never causes one.
type countingTransport struct {
	requests atomic.Int64
}

func (t *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.requests.Add(1)

	return http.DefaultTransport.RoundTrip(req)
}

// idp is a fake OIDC provider. It serves a JWKS document and, optionally, a
// discovery document pointing at it.
type idp struct {
	server *httptest.Server
	key    *rsa.PrivateKey

	jwksHits      atomic.Int64
	discoveryHits atomic.Int64
}

func newIDP(t *testing.T) *idp {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	i := &idp{key: key}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		i.discoveryHits.Add(1)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                i.issuer(),
			"jwks_uri":                              i.jwksURI(),
			"authorization_endpoint":                i.issuer() + "/authorize",
			"token_endpoint":                        i.issuer() + "/token",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		}))
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		i.jwksHits.Add(1)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{{
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"kid": testKeyID,
				"n":   base64.RawURLEncoding.EncodeToString(i.key.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(i.key.E)).Bytes()),
			}},
		}))
	})

	i.server = httptest.NewServer(mux)
	t.Cleanup(i.server.Close)

	return i
}

func (i *idp) issuer() string  { return i.server.URL }
func (i *idp) jwksURI() string { return i.server.URL + "/jwks" }

// token signs a token with the provider's key
func (i *idp) token(t *testing.T, issuer, audience string, expiry time.Time) string {
	t.Helper()

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   "user@example.com",
		Audience:  jwt.ClaimStrings{audience},
		ExpiresAt: jwt.NewNumericDate(expiry),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	})
	tok.Header["kid"] = testKeyID

	signed, err := tok.SignedString(i.key)
	require.NoError(t, err)

	return signed
}

// unsignedToken builds a token that is never expected to reach signature
// verification, so it doesn't need a real key
func unsignedToken(t *testing.T, issuer string) string {
	t.Helper()

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    issuer,
		Audience:  jwt.ClaimStrings{"whatever"},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString([]byte("not-a-real-key"))
	require.NoError(t, err)

	return signed
}

// TestValidate covers the trusted issuer lookup and the claim checks that
// follow it
func TestValidate(t *testing.T) {
	// Stands in for Temporal Cloud - a static JWKS URI, no discovery
	cloud := newIDP(t)
	// Stands in for a self-hosted deployment - JWKS found via discovery
	selfHosted := newIDP(t)

	const (
		cloudAudience      = "cloud-client-id"
		selfHostedAudience = "self-hosted-client-id"
	)

	issuers := map[string]auth.IssuerConfig{
		cloud.issuer(): {
			Issuer:   cloud.issuer(),
			Audience: cloudAudience,
			JWKSURI:  cloud.jwksURI(),
		},
		selfHosted.issuer(): {
			Issuer:   selfHosted.issuer(),
			Audience: selfHostedAudience,
		},
	}

	tests := []struct {
		name     string
		authType string
		token    func(t *testing.T) string
		// wantValid is true when the token is expected to pass validation
		wantValid bool
		// wantErr, when set, is the error the failure must wrap
		wantErr error
		// wantNoRequests asserts the token was rejected without the validator
		// making any outbound HTTP request
		wantNoRequests bool
	}{
		{
			name:     "valid cloud token",
			authType: bearer,
			token: func(t *testing.T) string {
				return cloud.token(t, cloud.issuer(), cloudAudience, time.Now().Add(time.Hour))
			},
			wantValid: true,
		},
		{
			name:     "valid self-hosted token",
			authType: bearer,
			token: func(t *testing.T) string {
				return selfHosted.token(t, selfHosted.issuer(), selfHostedAudience, time.Now().Add(time.Hour))
			},
			wantValid: true,
		},
		{
			name:     "untrusted issuer",
			authType: bearer,
			token: func(t *testing.T) string {
				return unsignedToken(t, "https://evil.example.com/")
			},
			wantErr: auth.ErrUntrustedIssuer,
			// Must be rejected before any key material is fetched
			wantNoRequests: true,
		},
		{
			name:     "valid signature but wrong audience",
			authType: bearer,
			token: func(t *testing.T) string {
				return cloud.token(t, cloud.issuer(), "some-other-client-id", time.Now().Add(time.Hour))
			},
			// go-oidc doesn't export a sentinel error for a bad audience
		},
		{
			name:     "expired token",
			authType: bearer,
			token: func(t *testing.T) string {
				return cloud.token(t, cloud.issuer(), cloudAudience, time.Now().Add(-time.Hour))
			},
		},
		{
			name:     "token signed by a different issuer's key",
			authType: bearer,
			token: func(t *testing.T) string {
				// Self-hosted key, but claiming to be from the cloud issuer
				return selfHosted.token(t, cloud.issuer(), cloudAudience, time.Now().Add(time.Hour))
			},
		},
		{
			name:     "no issuer claim",
			authType: bearer,
			token: func(t *testing.T) string {
				return unsignedToken(t, "")
			},
			wantErr:        auth.ErrNoIssuer,
			wantNoRequests: true,
		},
		{
			name:     "not a jwt",
			authType: bearer,
			token: func(*testing.T) string {
				return "definitely-not-a-jwt"
			},
			wantNoRequests: true,
		},
		{
			name:     "wrong auth type",
			authType: "Basic",
			token: func(t *testing.T) string {
				return cloud.token(t, cloud.issuer(), cloudAudience, time.Now().Add(time.Hour))
			},
			wantErr:        pkgauth.ErrInvalidAuthType,
			wantNoRequests: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// A fresh validator per test so that no key set is shared, and the
			// request count only reflects this test
			transport := new(countingTransport)
			validator := auth.NewValidator(issuers, &http.Client{Transport: transport, Timeout: time.Second * 10})

			err := validator.Validate(t.Context(), test.authType, test.token(t))

			if test.wantValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			if test.wantErr != nil {
				assert.ErrorIs(t, err, test.wantErr)
			}

			if test.wantNoRequests {
				assert.Zero(t, transport.requests.Load(), "no outbound request should have been made")
			}
		})
	}
}

// The cloud issuer uses its static JWKS URI, so it must never run discovery.
// The self-hosted issuer has no static URI, so it must.
func TestDiscovery(t *testing.T) {
	cloud := newIDP(t)
	selfHosted := newIDP(t)

	issuers := map[string]auth.IssuerConfig{
		cloud.issuer(): {
			Issuer:   cloud.issuer(),
			Audience: "cloud-client-id",
			JWKSURI:  cloud.jwksURI(),
		},
		selfHosted.issuer(): {
			Issuer:   selfHosted.issuer(),
			Audience: "self-hosted-client-id",
		},
	}

	validator := auth.NewValidator(issuers, nil)

	require.NoError(t, validator.Validate(
		t.Context(),
		bearer,
		cloud.token(t, cloud.issuer(), "cloud-client-id", time.Now().Add(time.Hour)),
	))
	assert.Zero(t, cloud.discoveryHits.Load(), "static JWKS URI should not trigger discovery")
	assert.Positive(t, cloud.jwksHits.Load(), "static JWKS URI should have been fetched")

	require.NoError(t, validator.Validate(
		t.Context(),
		bearer,
		selfHosted.token(t, selfHosted.issuer(), "self-hosted-client-id", time.Now().Add(time.Hour)),
	))
	assert.Equal(t, int64(1), selfHosted.discoveryHits.Load(), "discovery should have run once")
	assert.Positive(t, selfHosted.jwksHits.Load(), "discovered JWKS URI should have been fetched")
}

// Verifiers - and therefore discovery - should only be built once per issuer
func TestVerifierIsCached(t *testing.T) {
	selfHosted := newIDP(t)

	validator := auth.NewValidator(map[string]auth.IssuerConfig{
		selfHosted.issuer(): {
			Issuer:   selfHosted.issuer(),
			Audience: "self-hosted-client-id",
		},
	}, nil)

	for i := range 3 {
		token := selfHosted.token(t, selfHosted.issuer(), "self-hosted-client-id", time.Now().Add(time.Hour))
		require.NoError(t, validator.Validate(t.Context(), bearer, token), fmt.Sprintf("attempt %d", i))
	}

	assert.Equal(t, int64(1), selfHosted.discoveryHits.Load(), "discovery should only run once per issuer")
}

// The hardcoded Temporal Cloud entry must keep pointing at Cloud's static JWKS
// endpoint
func TestTemporalCloudIsTrusted(t *testing.T) {
	assert.Equal(t, pkgauth.TemporalIssuerURL, auth.TemporalCloudJWKSURI)

	// An untrusted issuer is still rejected by the default, hardcoded map
	err := auth.TrustedIssuers(bearer, unsignedToken(t, "https://evil.example.com/"))
	assert.ErrorIs(t, err, auth.ErrUntrustedIssuer)
}

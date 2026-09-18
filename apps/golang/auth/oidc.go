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

// Package auth validates the JSON Web Tokens that the Temporal Web UI forwards
// to the codec server in the "Authorization" header.
//
// Both Temporal Cloud and self-hosted Temporal deployments sign their tokens
// with an OIDC provider, but each deployment uses a different one. Rather than
// assume every token was minted by Temporal Cloud, the token's issuer is read
// first and looked up against a list of issuers this server trusts.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	pkgauth "github.com/mrsimonemms/temporal-codec-server/packages/golang/auth"
)

// Errors returned when a token cannot be trusted. These are deliberately vague
// about why - the detail is for the logs, not the caller.
var (
	ErrUntrustedIssuer = errors.New("token issuer is not trusted")
	ErrNoIssuer        = errors.New("token has no issuer claim")
)

// Temporal Cloud's OIDC provider. Cloud's JWKS endpoint is fixed and well
// known, so it's declared statically rather than discovered.
const (
	TemporalCloudIssuer  = "https://login.tmprl.cloud/"
	TemporalCloudJWKSURI = pkgauth.TemporalIssuerURL
	// The audience Temporal Cloud mints Web UI tokens for
	TemporalCloudAudience = "https://saas-api.tmprl.cloud"
)

// How long to wait for a JWKS or OIDC discovery document
const discoveryTimeout = 10 * time.Second

// IssuerConfig describes a single OIDC provider whose tokens this codec server
// will accept.
type IssuerConfig struct {
	// Issuer is the exact value the token's "iss" claim must contain. It is
	// also the URL that OIDC discovery is run against when JWKSURI is empty.
	Issuer string

	// Audience is the OIDC client ID the token must have been issued for. The
	// token's "aud" claim must contain this value.
	Audience string

	// JWKSURI is an optional static JWKS endpoint. When set, the signing keys
	// are fetched from here directly. When empty, the endpoint is discovered
	// from the issuer's .well-known/openid-configuration document.
	JWKSURI string
}

// The issuers this codec server trusts, keyed by issuer URL.
var trustedIssuers = map[string]IssuerConfig{
	TemporalCloudIssuer: {
		Issuer:   TemporalCloudIssuer,
		Audience: TemporalCloudAudience,
		JWKSURI:  TemporalCloudJWKSURI,
	},
	// A self-hosted deployment's own OIDC provider. No JWKS URI is set, so the
	// signing keys are discovered from the issuer.
	"https://oidc.simonemms.com": {
		Issuer:   "https://sso.example.com/realms/temporal",
		Audience: "2f86f549-0489-4d42-8565-9fe7bce1642,",
	},
}

// Validator verifies tokens against a fixed set of trusted OIDC issuers.
//
// Verifiers are built lazily and then cached, one per issuer, so that OIDC
// discovery runs at most once per issuer and the underlying key set can manage
// its own JWKS caching and refreshing.
type Validator struct {
	issuers map[string]IssuerConfig
	client  *http.Client

	// The context the cached key sets make their HTTP requests with. It has to
	// outlive the request that created them, so it is deliberately not tied to
	// any single call to Validate.
	ctx context.Context

	mu        sync.Mutex
	verifiers map[string]*oidc.IDTokenVerifier
}

// NewValidator builds a Validator for the given issuers. A nil client uses a
// default one.
func NewValidator(issuers map[string]IssuerConfig, client *http.Client) *Validator {
	if client == nil {
		client = &http.Client{Timeout: discoveryTimeout}
	}

	return &Validator{
		issuers:   issuers,
		client:    client,
		ctx:       oidc.ClientContext(context.Background(), client),
		verifiers: map[string]*oidc.IDTokenVerifier{},
	}
}

// Validate checks that the token was issued by a trusted issuer and that its
// signature, expiry, audience and issuer are all valid.
func (v *Validator) Validate(ctx context.Context, authType, token string) error {
	if !strings.EqualFold(authType, "Bearer") {
		return pkgauth.ErrInvalidAuthType
	}

	// Read the issuer before doing anything else. This is an unverified read of
	// an untrusted token, so the value is only ever used to decide which set of
	// signing keys the token should be checked against - never to trust it.
	issuer, err := unverifiedIssuer(token)
	if err != nil {
		return err
	}

	// Reject unknown issuers here, before any key material is fetched. A token
	// from an issuer we don't trust must never cause an outbound request.
	cfg, ok := v.issuers[issuer]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUntrustedIssuer, issuer)
	}

	verifier, err := v.verifier(cfg)
	if err != nil {
		return err
	}

	if _, err := verifier.Verify(oidc.ClientContext(ctx, v.client), token); err != nil {
		return fmt.Errorf("error verifying token: %w", err)
	}

	return nil
}

// verifier returns the cached verifier for an issuer, building it - and running
// OIDC discovery, if needed - on first use.
func (v *Validator) verifier(cfg IssuerConfig) (*oidc.IDTokenVerifier, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if verifier, ok := v.verifiers[cfg.Issuer]; ok {
		return verifier, nil
	}

	oidcCfg := &oidc.Config{
		ClientID: cfg.Audience,
		// An issuer with no audience configured can't have its audience checked
		SkipClientIDCheck: cfg.Audience == "",
	}

	var verifier *oidc.IDTokenVerifier
	if cfg.JWKSURI != "" {
		// A static JWKS endpoint - no discovery required
		verifier = oidc.NewVerifier(cfg.Issuer, oidc.NewRemoteKeySet(v.ctx, cfg.JWKSURI), oidcCfg)
	} else {
		ctx, cancel := context.WithTimeout(v.ctx, discoveryTimeout)
		defer cancel()

		// Discovery also asserts that the document's issuer matches cfg.Issuer
		provider, err := oidc.NewProvider(ctx, cfg.Issuer)
		if err != nil {
			return nil, fmt.Errorf("error discovering oidc provider %s: %w", cfg.Issuer, err)
		}

		verifier = provider.VerifierContext(v.ctx, oidcCfg)
	}

	v.verifiers[cfg.Issuer] = verifier

	return verifier, nil
}

// unverifiedIssuer reads the "iss" claim without checking the signature. The
// token is still untrusted at this point.
func unverifiedIssuer(token string) (string, error) {
	claims := new(jwt.RegisteredClaims)
	if _, _, err := jwt.NewParser().ParseUnverified(token, claims); err != nil {
		return "", fmt.Errorf("error parsing unverified token: %w", err)
	}

	if claims.Issuer == "" {
		return "", ErrNoIssuer
	}

	return claims.Issuer, nil
}

var defaultValidator = NewValidator(trustedIssuers, nil)

// TrustedIssuers validates a bearer token against the trusted OIDC issuers. It
// satisfies auth.MiddlewareAuthFunction.
func TrustedIssuers(authType, token string) error {
	return defaultValidator.Validate(context.Background(), authType, token)
}

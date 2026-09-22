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

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/converter"
)

const cloudOrigin = "https://cloud.temporal.io"

// The documented route shape is <endpoint>/encode and <endpoint>/decode, where
// the endpoint may itself carry a namespace segment. Both the Web UI codec
// config and the CLI's --codec-endpoint support a "{namespace}" placeholder,
// e.g. --codec-endpoint 'http://localhost:8081/{namespace}'.
//
// https://docs.temporal.io/production-deployment/data-encryption#cli
//
// Only POST is part of the protocol. Anything else must be refused.
func TestRoutes_RejectNonPostMethods(t *testing.T) {
	methods := []string{
		// GET and HEAD are the two the health-check probe answers, so they are
		// the ones at risk of being swallowed before reaching a codec route.
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	}
	targets := []string{pathEncode, pathDecode, pathNamespacedEncode, pathNamespacedDecode}

	app := newAESApp(t)

	for _, method := range methods {
		for _, target := range targets {
			t.Run(method+" "+target, func(t *testing.T) {
				res := do(t, app, request{method: method, target: target})

				assert.NotEqual(t, http.StatusOK, res.status,
					"%s %s must not be served as if it were a codec request", method, target)
			})
		}
	}
}

// Fiber answers a known path with an unknown method with 405 and an Allow
// header naming the methods that are registered.
func TestRoutes_MethodNotAllowed(t *testing.T) {
	app := newAESApp(t)

	res := do(t, app, request{method: http.MethodPut, target: pathDecode})

	assert.Equal(t, http.StatusMethodNotAllowed, res.status)
	assert.Equal(t, http.MethodPost, res.headers.Get("Allow"))
}

// Paths that are neither the documented routes nor a configured namespace must
// 404 rather than falling through to a codec.
func TestRoutes_UnknownPaths(t *testing.T) {
	tests := []string{
		"//decode",
		"/a/b/decode",
		"/decode/",
		"/encode/extra",
		"/default/download",
	}

	app := newAESApp(t)

	for _, target := range tests {
		t.Run(target, func(t *testing.T) {
			res := post(t, app, target, `{"payloads":[]}`)

			assert.Equal(t, http.StatusNotFound, res.status, res.body)
		})
	}
}

// The health-check probe is mounted on its own endpoints. If it is ever
// mounted globally again it will answer every GET and HEAD, including the
// codec routes, which is what TestRoutes_RejectNonPostMethods guards. This
// test is the other half of that pair: the probe must still work.
func TestRoutes_HealthEndpointsStillServed(t *testing.T) {
	app := newAESApp(t)

	for _, target := range []string{healthcheck.LivenessEndpoint, healthcheck.ReadinessEndpoint} {
		t.Run(target, func(t *testing.T) {
			res := do(t, app, request{method: http.MethodGet, target: target})

			assert.Equal(t, http.StatusOK, res.status)
			assert.Equal(t, "OK", res.body)
		})
	}
}

// #################### //
// Namespace resolution //
// #################### //

// The namespace in the route selects the codec chain.
func TestNamespace_RouteSelectsCodec(t *testing.T) {
	primary := &recordingCodec{name: codecPrimary}
	secondary := &recordingCodec{name: codecSecondary}

	app := newTestApp(t, &Config{
		Encoders: map[string][]converter.PayloadCodec{
			defaultNamespace: {primary},
			otherNamespace:   {secondary},
		},
	})

	res := post(t, app, "/"+otherNamespace+pathDecode, payloadsJSON(t, plainPayload(t, "hello")))
	require.Equal(t, http.StatusOK, res.status, res.body)

	payloads := parsePayloads(t, res.body)
	require.Len(t, payloads, 1)
	assert.Equal(t, codecSecondary, string(payloads[0].GetMetadata()["decoded-by"]))
	assert.Zero(t, primary.decodeCallCount())
}

// The Web UI does not use a namespaced route; it sends X-Namespace instead. A
// namespace in the route is authoritative and cannot be overridden by it.
//
// https://docs.temporal.io/production-deployment/data-encryption#headers
func TestNamespace_HeaderHandling(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		namespace string
		wantCodec string
	}{
		{
			name:      "header selects the codec",
			target:    pathDecode,
			namespace: otherNamespace,
			wantCodec: codecSecondary,
		},
		{
			name:      "route wins over the header",
			target:    pathNamespacedDecode,
			namespace: otherNamespace,
			wantCodec: codecPrimary,
		},
		{
			name:      "header naming the default namespace",
			target:    pathDecode,
			namespace: defaultNamespace,
			wantCodec: codecPrimary,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t, &Config{
				Encoders: map[string][]converter.PayloadCodec{
					defaultNamespace: {&recordingCodec{name: codecPrimary}},
					otherNamespace:   {&recordingCodec{name: codecSecondary}},
				},
			})

			res := post(t, app, test.target, payloadsJSON(t, plainPayload(t, "hello")),
				map[string]string{headerNamespace: test.namespace})
			require.Equal(t, http.StatusOK, res.status, res.body)

			payloads := parsePayloads(t, res.body)
			require.Len(t, payloads, 1)
			assert.Equal(t, test.wantCodec, string(payloads[0].GetMetadata()["decoded-by"]))
		})
	}
}

// No usable namespace anywhere means the default namespace. HTTP strips
// surrounding whitespace from header values, so a blank-looking X-Namespace
// arrives empty and needs no trimming of its own.
func TestNamespace_FallsBackToDefault(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
	}{
		{name: "no header"},
		{name: "empty header", headers: map[string]string{headerNamespace: ""}},
		{name: "whitespace header", headers: map[string]string{headerNamespace: "   "}},
		{name: "padded default", headers: map[string]string{headerNamespace: "  default  "}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			codec := &recordingCodec{name: codecPrimary}
			app := newTestApp(t, &Config{
				Encoders: map[string][]converter.PayloadCodec{
					defaultNamespace: {codec},
				},
			})

			res := post(t, app, pathDecode, payloadsJSON(t, plainPayload(t, "hello")), test.headers)
			require.Equal(t, http.StatusOK, res.status, res.body)

			payloads := parsePayloads(t, res.body)
			require.Len(t, payloads, 1)
			assert.Equal(t, codecPrimary, string(payloads[0].GetMetadata()["decoded-by"]))
		})
	}
}

// A namespace with no configured codec must not be served by another
// namespace's codec.
func TestNamespace_UnknownNamespaceIsRejected(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		headers map[string]string
	}{
		{
			name:   "unknown namespace in route",
			target: "/not-configured/decode",
		},
		{
			name:   "namespace is case sensitive",
			target: "/DEFAULT/decode",
		},
		{
			name:    "unknown namespace in header",
			target:  pathDecode,
			headers: map[string]string{headerNamespace: "not-configured"},
		},
		{
			name:   "unknown namespace in encode route",
			target: "/" + otherNamespace + pathEncode,
		},
		{
			name:    "unknown namespace in header on encode",
			target:  pathEncode,
			headers: map[string]string{headerNamespace: otherNamespace},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			codec := &recordingCodec{name: codecPrimary}
			app := newTestApp(t, &Config{
				Encoders: map[string][]converter.PayloadCodec{
					defaultNamespace: {codec},
				},
			})

			res := post(t, app, test.target, payloadsJSON(t, plainPayload(t, "hello")), test.headers)

			assert.Equal(t, http.StatusNotFound, res.status, res.body)
			assert.Zero(t, codec.decodeCallCount(),
				"a request for an unconfigured namespace must not reach another namespace's codec")
		})
	}
}

// With no codecs configured at all there is nothing to serve.
func TestNamespace_NoEncodersConfigured(t *testing.T) {
	app := newTestApp(t, &Config{Encoders: map[string][]converter.PayloadCodec{}})

	res := post(t, app, pathDecode, `{"payloads":[]}`)

	assert.Equal(t, http.StatusNotFound, res.status)
}

// The three ways of naming a namespace are independent of each other. Handed
// only a named namespace, the router serves that namespace by route and by
// header but has no default to fall back on. config.GetEncoders is what
// guarantees a default entry always exists in a real deployment; the router
// itself makes no such assumption.
func TestNamespace_OnlyNamedNamespaceConfigured(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		headers    map[string]string
		wantStatus int
	}{
		{
			name:       "named route is served",
			target:     "/" + otherNamespace + pathDecode,
			wantStatus: http.StatusOK,
		},
		{
			name:       "named header is served",
			target:     pathDecode,
			headers:    map[string]string{headerNamespace: otherNamespace},
			wantStatus: http.StatusOK,
		},
		{
			name:       "no namespace has no default to fall back to",
			target:     pathDecode,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t, &Config{
				Encoders: map[string][]converter.PayloadCodec{
					otherNamespace: {&recordingCodec{name: codecSecondary}},
				},
			})

			res := post(t, app, test.target, payloadsJSON(t, plainPayload(t, "hello")), test.headers)

			assert.Equal(t, test.wantStatus, res.status, res.body)
		})
	}
}

// Namespace resolution is shared by both endpoints, so X-Namespace must select
// the codec on /encode exactly as it does on /decode.
func TestNamespace_HeaderAppliesToEncode(t *testing.T) {
	app := newTestApp(t, &Config{
		Encoders: map[string][]converter.PayloadCodec{
			defaultNamespace: {&recordingCodec{name: codecPrimary}},
			otherNamespace:   {&recordingCodec{name: codecSecondary}},
		},
	})

	res := post(t, app, pathEncode, payloadsJSON(t, plainPayload(t, "hello")),
		map[string]string{headerNamespace: otherNamespace})
	require.Equal(t, http.StatusOK, res.status, res.body)

	payloads := parsePayloads(t, res.body)
	require.Len(t, payloads, 1)
	assert.Equal(t, codecSecondary, string(payloads[0].GetMetadata()["encoded-by"]))
}

// ############ //
// Request body //
// ############ //

// The Web UI always sends Content-Type: application/json. The server must
// accept it (and, as it happens, does not require it).
func TestRequest_ContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{name: "documented", contentType: fiber.MIMEApplicationJSON},
		{name: "with charset", contentType: "application/json; charset=utf-8"},
	}

	app := newAESApp(t)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := do(t, app, request{
				target:  pathDecode,
				body:    payloadsJSON(t, plainPayload(t, "hello")),
				headers: map[string]string{fiber.HeaderContentType: test.contentType},
			})

			assert.Equal(t, http.StatusOK, res.status, res.body)
		})
	}
}

// #### //
// CORS //
// #### //

func corsConfig() *Config {
	return &Config{
		EnableCORS:     true,
		CORSAllowCreds: true,
		CORSOrigins:    []string{cloudOrigin},
		Encoders: map[string][]converter.PayloadCodec{
			defaultNamespace: {&recordingCodec{name: codecPrimary}},
		},
	}
}

// The browser preflights every codec request. Temporal documents the minimum
// response as Access-Control-Allow-Origin, -Methods and -Headers, with
// X-Namespace, Content-Type and Authorization allowed.
//
// https://docs.temporal.io/production-deployment/data-encryption#cors
func TestCORS_Preflight(t *testing.T) {
	targets := []string{pathEncode, pathDecode, pathNamespacedEncode, pathNamespacedDecode}

	app := newTestApp(t, corsConfig())

	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			res := do(t, app, request{
				method: http.MethodOptions,
				target: target,
				headers: map[string]string{
					headerOrigin:               cloudOrigin,
					headerRequestMethod:        http.MethodPost,
					headerRequestHeadersHeader: "content-type,x-namespace,authorization",
				},
			})

			assert.Equal(t, http.StatusNoContent, res.status)
			assert.Equal(t, cloudOrigin, res.headers.Get(headerAllowOrigin))
			assert.Equal(t, "true", res.headers.Get(headerAllowCredentials))

			assert.Contains(t, res.headers.Get(headerAllowMethods), http.MethodPost)

			allowHeaders := res.headers.Get(headerAllowHeaders)
			for _, header := range []string{"Content-Type", headerNamespace, "Authorization"} {
				assert.Contains(t, allowHeaders, header)
			}
		})
	}
}

// Temporal documents the minimum allowed methods as "POST, GET, OPTIONS".
func TestCORS_PreflightAllowsDocumentedMethods(t *testing.T) {
	app := newTestApp(t, corsConfig())

	res := do(t, app, request{
		method: http.MethodOptions,
		target: pathDecode,
		headers: map[string]string{
			headerOrigin:        cloudOrigin,
			headerRequestMethod: http.MethodPost,
		},
	})

	allowMethods := res.headers.Get(headerAllowMethods)
	for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodOptions} {
		assert.Contains(t, allowMethods, method)
	}
}

// The actual (non-preflight) request must also carry the CORS response
// headers, or the browser discards the response.
func TestCORS_ActualRequest(t *testing.T) {
	app := newTestApp(t, corsConfig())

	res := post(t, app, pathDecode, payloadsJSON(t, plainPayload(t, "hello")),
		map[string]string{headerOrigin: cloudOrigin})

	require.Equal(t, http.StatusOK, res.status, res.body)
	assert.Equal(t, cloudOrigin, res.headers.Get(headerAllowOrigin))
	assert.Equal(t, "true", res.headers.Get(headerAllowCredentials))
}

// A browser can only read a response it is allowed to read. The CORS headers
// must therefore survive the error paths too, or an operator debugging an
// unknown-namespace 404 in the Web UI sees an opaque network failure instead
// of the status the server actually sent.
func TestCORS_HeadersPresentOnErrorResponses(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		body       string
		wantStatus int
	}{
		{
			name:       "unknown namespace",
			target:     "/not-configured" + pathDecode,
			body:       `{"payloads":[]}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "malformed body",
			target:     pathDecode,
			body:       notJSON,
			wantStatus: http.StatusBadRequest,
		},
	}

	app := newTestApp(t, corsConfig())

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := post(t, app, test.target, test.body, map[string]string{headerOrigin: cloudOrigin})

			require.Equal(t, test.wantStatus, res.status, res.body)
			assert.Equal(t, cloudOrigin, res.headers.Get(headerAllowOrigin))
			assert.Equal(t, "true", res.headers.Get(headerAllowCredentials))
		})
	}
}

// An origin that is not configured must not be granted access.
func TestCORS_DisallowedOrigin(t *testing.T) {
	app := newTestApp(t, corsConfig())

	res := post(t, app, pathDecode, payloadsJSON(t, plainPayload(t, "hello")),
		map[string]string{headerOrigin: "https://evil.example.com"})

	assert.Empty(t, res.headers.Get(headerAllowOrigin))
}

// With CORS off there is no preflight handler, so a browser cannot call the
// codec server cross-origin at all.
func TestCORS_Disabled(t *testing.T) {
	app := newAESApp(t)

	preflight := do(t, app, request{
		method: http.MethodOptions,
		target: pathDecode,
		headers: map[string]string{
			headerOrigin:        cloudOrigin,
			headerRequestMethod: http.MethodPost,
		},
	})
	assert.Equal(t, http.StatusMethodNotAllowed, preflight.status)

	actual := post(t, app, pathDecode, `{"payloads":[]}`, map[string]string{headerOrigin: cloudOrigin})
	assert.Empty(t, actual.headers.Get(headerAllowOrigin))
}

// Enabling credentials without naming any origins leaves Fiber's wildcard
// default in place, which is an illegal combination. The router panics while
// being built rather than starting up in a broken state.
func TestCORS_CredentialsWithoutOriginsPanics(t *testing.T) {
	assert.Panics(t, func() {
		newTestApp(t, &Config{
			EnableCORS:     true,
			CORSAllowCreds: true,
			Encoders: map[string][]converter.PayloadCodec{
				defaultNamespace: {&recordingCodec{name: codecPrimary}},
			},
		})
	})
}

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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	// defaultNamespace mirrors client.DefaultNamespace, which is what the router
	// falls back to when no namespace is supplied by route or header.
	defaultNamespace = "default"

	// externalStorageReferenceMessageType is the proto message name the Go SDK
	// uses to mark a payload as an External Storage reference.
	externalStorageReferenceMessageType = "temporal.api.sdk.v1.ExternalStorageReference"

	// fiberJSONContentType is the exact Content-Type the codec handler sets on
	// a successful response.
	fiberJSONContentType = "application/json"

	// The four codec routes under test.
	pathEncode           = "/encode"
	pathDecode           = "/decode"
	pathNamespacedEncode = "/default/encode"
	pathNamespacedDecode = "/default/decode"

	// otherNamespace is a second, non-default namespace, in the shape Temporal
	// Cloud uses.
	otherNamespace = "myapp-dev.acctid123"

	headerNamespace            = "X-Namespace"
	headerOrigin               = "Origin"
	headerRequestMethod        = "Access-Control-Request-Method"
	headerAllowOrigin          = "Access-Control-Allow-Origin"
	headerAllowMethods         = "Access-Control-Allow-Methods"
	headerAllowHeaders         = "Access-Control-Allow-Headers"
	headerAllowCredentials     = "Access-Control-Allow-Credentials"
	headerRequestHeadersHeader = "Access-Control-Request-Headers"

	codecPrimary   = "primary"
	codecSecondary = "secondary"

	// The two codec directions, used both as recordingCodec labels and as the
	// metadata keys it tags payloads with.
	opEncode = "encode"
	opDecode = "decode"

	// pathDecodePreserveRefs is the URL the Web UI uses for Event History
	// decoding when External Storage is in play.
	pathDecodePreserveRefs = pathDecode + "?preserveStorageRefs=true"

	// notJSON is a body that is not parseable as JSON at all.
	notJSON = "not json"
)

// testKeys is a valid AES key set. The key must be exactly 32 bytes for AES-256.
var testKeys = aes.Keys{
	{ID: "key0", Key: "passphrasewhichneedstobe32bytes!"},
}

// recordingCodec is a PayloadCodec that records how it was invoked and tags
// every payload it touches so that tests can prove which codec chain ran.
//
// A codec handler is shared by every request, so a codec can be called from
// several goroutines at once. The bookkeeping is therefore mutex guarded; a
// plain counter races under -race.
type recordingCodec struct {
	name string
	// order, when set, is appended to on every call so that tests can assert
	// the order in which a chain of codecs ran.
	order *[]string

	mu          sync.Mutex
	encodeCalls int
	decodeCalls int
}

func (c *recordingCodec) Encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	c.record(opEncode)

	return c.tag(payloads, "encoded-by"), nil
}

func (c *recordingCodec) Decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	c.record(opDecode)

	return c.tag(payloads, "decoded-by"), nil
}

func (c *recordingCodec) record(operation string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if operation == opEncode {
		c.encodeCalls++
	} else {
		c.decodeCalls++
	}

	if c.order != nil {
		*c.order = append(*c.order, c.name+":"+operation)
	}
}

func (c *recordingCodec) encodeCallCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.encodeCalls
}

func (c *recordingCodec) decodeCallCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.decodeCalls
}

func (c *recordingCodec) tag(payloads []*commonpb.Payload, key string) []*commonpb.Payload {
	result := make([]*commonpb.Payload, len(payloads))
	for i, p := range payloads {
		metadata := map[string][]byte{}
		for k, v := range p.GetMetadata() {
			metadata[k] = v
		}
		metadata[key] = []byte(c.name)

		result[i] = &commonpb.Payload{Metadata: metadata, Data: p.GetData()}
	}
	return result
}

// failingCodec always errors, to exercise the codec failure paths.
type failingCodec struct{}

func (failingCodec) Encode([]*commonpb.Payload) ([]*commonpb.Payload, error) {
	return nil, fmt.Errorf("encode exploded")
}

func (failingCodec) Decode([]*commonpb.Payload) ([]*commonpb.Payload, error) {
	return nil, fmt.Errorf("decode exploded")
}

// newTestApp builds a router with the given config on a throwaway Fiber app.
func newTestApp(t *testing.T, cfg *Config) *fiber.App {
	t.Helper()

	app := fiber.New()
	New(app, cfg)

	return app
}

// newAESApp builds a router whose default namespace is served by the real AES
// codec used in production.
func newAESApp(t *testing.T) *fiber.App {
	t.Helper()

	return newTestApp(t, &Config{
		Encoders: map[string][]converter.PayloadCodec{
			defaultNamespace: {aes.NewPayloadCodec(testKeys)},
		},
	})
}

type request struct {
	method  string
	target  string
	body    string
	headers map[string]string
}

type response struct {
	status  int
	headers http.Header
	body    string
}

// do issues req against app. Requests default to the method, path and
// Content-Type that the Web UI and CLI use.
func do(t *testing.T, app *fiber.App, req request) response {
	t.Helper()

	if req.method == "" {
		req.method = http.MethodPost
	}

	httpReq := httptest.NewRequestWithContext(
		t.Context(), req.method, req.target, strings.NewReader(req.body),
	)
	httpReq.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	for k, v := range req.headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := app.Test(httpReq)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = resp.Body.Close()
	})

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return response{
		status:  resp.StatusCode,
		headers: resp.Header,
		body:    string(body),
	}
}

// post is the common case: a POST of a payload envelope.
func post(t *testing.T, app *fiber.App, target, body string, headers ...map[string]string) response {
	t.Helper()

	req := request{target: target, body: body}
	if len(headers) > 0 {
		req.headers = headers[0]
	}

	return do(t, app, req)
}

// payloadsJSON builds the documented request envelope. The metadata values and
// data are base64 encoded on the wire, which protojson handles for us.
func payloadsJSON(t *testing.T, payloads ...*commonpb.Payload) string {
	t.Helper()

	bs, err := protojson.Marshal(&commonpb.Payloads{Payloads: payloads})
	require.NoError(t, err)

	return string(bs)
}

// plainPayload builds an unencoded json/plain payload, as a Temporal client
// would produce before any codec runs.
func plainPayload(t *testing.T, value any) *commonpb.Payload {
	t.Helper()

	p, err := converter.GetDefaultDataConverter().ToPayload(value)
	require.NoError(t, err)

	return p
}

// storageRefPayload builds an External Storage reference payload in the format
// the current Go SDK writes and recognises: encoding=json/protobuf with a
// messageType of temporal.api.sdk.v1.ExternalStorageReference.
//
// See go.temporal.io/sdk/internal/extstore.IsStorageReference.
func storageRefPayload(driver string) *commonpb.Payload {
	return &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding:    []byte(converter.MetadataEncodingProtoJSON),
			converter.MetadataMessageType: []byte(externalStorageReferenceMessageType),
		},
		Data: fmt.Appendf(nil, `{"driverName":%q}`, driver),
	}
}

// parsePayloads reads a codec server response envelope.
func parsePayloads(t *testing.T, body string) []*commonpb.Payload {
	t.Helper()

	var payloads commonpb.Payloads
	require.NoError(t, protojson.Unmarshal([]byte(body), &payloads), "response body: %s", body)

	return payloads.GetPayloads()
}

// rawEnvelope reads the response as generic JSON, for assertions about the
// shape of the envelope itself rather than its contents.
func rawEnvelope(t *testing.T, body string) map[string]any {
	t.Helper()

	out := map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(body), &out), "response body: %s", body)

	return out
}

func encodingOf(p *commonpb.Payload) string {
	return string(p.GetMetadata()[converter.MetadataEncoding])
}

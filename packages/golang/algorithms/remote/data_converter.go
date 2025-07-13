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

package remote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"time"

	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

// This is a very silly idea. This encrypter connects to a remote
// Temporal Codec Server and encrypts everything over that. This
// will be slow.
//
// Very slow. So use this as what _CAN_ be done, not what _SHOULD_
// be done
//
// And development. It's (probably) fine for development.
type remote struct {
	url     string
	headers map[string]string
	client  *http.Client
}

func (r *remote) exec(endpoint string, payloads []*common.Payload) ([]*common.Payload, error) {
	input, err := json.Marshal(common.Payloads{
		Payloads: payloads,
	})
	if err != nil {
		return nil, fmt.Errorf("error converting payload to json: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s", r.url, endpoint), bytes.NewBuffer(input))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	for k, v := range r.headers {
		req.Header.Add(k, v)
	}

	res, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling encoding endpoint: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid data returned")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %w", err)
	}

	output := common.Payloads{}
	if err := json.Unmarshal(body, &output); err != nil {
		return nil, fmt.Errorf("error converting payloads: %w", err)
	}

	return output.GetPayloads(), nil
}

func (r *remote) Decode(payloads []*common.Payload) ([]*common.Payload, error) {
	return r.exec("decode", payloads)
}

func (r *remote) Encode(payloads []*common.Payload) ([]*common.Payload, error) {
	return r.exec("encode", payloads)
}

func DataConverter(url string, customHeaders ...map[string]string) converter.DataConverter {
	headers := map[string]string{}
	for _, c := range customHeaders {
		maps.Copy(headers, c)
	}

	return converter.NewCodecDataConverter(converter.GetDefaultDataConverter(), &remote{
		url:     strings.TrimSuffix(url, "/"), // Trim trailing slash
		headers: headers,
		client: &http.Client{
			Timeout: time.Second * 5,
		},
	})
}

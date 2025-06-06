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

package golang

import (
	"github.com/golang/snappy"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

var DataConverter = NewDataConverter(converter.GetDefaultDataConverter())

type Codec struct{}

func (*Codec) Decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))
	for i, p := range payloads {
		// Only if it's our encoding
		if string(p.Metadata[converter.MetadataEncoding]) != "binary/snappy" {
			result[i] = p
			continue
		}
		// Uncompress
		b, err := snappy.Decode(nil, p.Data)
		if err != nil {
			return payloads, err
		}
		// Unmarshal proto
		result[i] = &commonpb.Payload{}
		err = result[i].Unmarshal(b)
		if err != nil {
			return payloads, err
		}
	}

	return result, nil
}

func (e *Codec) Encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))
	for i, p := range payloads {
		// Marshal proto
		origBytes, err := p.Marshal()
		if err != nil {
			return payloads, err
		}
		// Compress
		b := snappy.Encode(nil, origBytes)
		result[i] = &commonpb.Payload{
			Metadata: map[string][]byte{converter.MetadataEncoding: []byte("binary/snappy")},
			Data:     b,
		}
	}

	return result, nil
}

func NewPayloadCodec() converter.PayloadCodec {
	return &Codec{}
}

// NewDataConverter creates a new data converter that wraps the converter
func NewDataConverter(underlying converter.DataConverter) converter.DataConverter {
	return converter.NewCodecDataConverter(underlying, NewPayloadCodec())
}

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

package external

import (
	"fmt"

	"github.com/google/uuid"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

const (
	ExternalMimeType = "binary/external"
	MetadataTypeID   = "database-type-id"
)

type codec struct {
	connection Connection
}

// Decode implements [converter.PayloadCodec].
func (c *codec) Decode(payloads []*common.Payload) ([]*common.Payload, error) {
	result := make([]*common.Payload, len(payloads))
	for i, p := range payloads {
		// Only if it's our encoding
		if string(p.Metadata[converter.MetadataEncoding]) != ExternalMimeType {
			result[i] = p
			continue
		}

		// And if it's this database
		targetKey := string(p.Metadata[MetadataTypeID])
		if string(targetKey) != c.connection.GetTypeID() {
			result[i] = p
			continue
		}

		key, err := uuid.Parse(string(p.Data))
		if err != nil {
			return nil, fmt.Errorf("error parsing database key to uuid: %w", err)
		}

		data, err := c.connection.Get(key)
		if err != nil {
			return nil, fmt.Errorf("error getting data from database: %w", err)
		}

		// Unmarshal proto
		result[i] = &common.Payload{}
		err = result[i].Unmarshal(data)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Encode implements [converter.PayloadCodec].
func (c *codec) Encode(payloads []*common.Payload) ([]*common.Payload, error) {
	result := make([]*common.Payload, len(payloads))
	for i, p := range payloads {
		// Marshal proto
		origBytes, err := p.Marshal()
		if err != nil {
			return payloads, err
		}

		key := uuid.New()

		if err := c.connection.Save(key, origBytes); err != nil {
			return nil, fmt.Errorf("error encoding data to database: %w", err)
		}

		result[i] = &common.Payload{
			Metadata: map[string][]byte{
				converter.MetadataEncoding: []byte(ExternalMimeType),
				MetadataTypeID:             []byte(c.connection.GetTypeID()),
			},
			Data: []byte(key.String()),
		}
	}

	return result, nil
}

func DataConverter(connection Connection) converter.DataConverter {
	return NewDataConverter(converter.GetDefaultDataConverter(), connection)
}

func NewPayloadCodec(connection Connection) converter.PayloadCodec {
	return &codec{connection: connection}
}

// NewDataConverter creates a new data converter that wraps the converter
func NewDataConverter(underlying converter.DataConverter, connection Connection) converter.DataConverter {
	return converter.NewCodecDataConverter(underlying, NewPayloadCodec(connection))
}

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

package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"go.temporal.io/api/common/v1"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

const (
	AESMimeType   = "binary/encrypted"
	MetadataKeyID = "encryption-key-id"
)

type codec struct {
	// Encoding strings in key/value format - keys are public refs
	// which allow keys to be refreshed
	keys Keys
}

// Decode implements converter.PayloadCodec.
func (c *codec) Decode(payloads []*common.Payload) ([]*common.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))
	for i, p := range payloads {
		// Only if it's our encoding
		if string(p.Metadata[converter.MetadataEncoding]) != AESMimeType {
			result[i] = p
			continue
		}

		// Iterate over the keys to find the one used
		var key *Key
		targetKey := string(p.Metadata[MetadataKeyID])
		if targetKey == "" {
			return nil, fmt.Errorf("no key id provided")
		}
		for _, k := range c.keys {
			if targetKey == k.ID {
				key = &k
			}
		}
		if key == nil {
			return nil, fmt.Errorf("unknown encryption key: %s", targetKey)
		}

		gcm, err := c.newCipher(*key)
		if err != nil {
			return nil, err
		}

		ciphertext := p.Data

		nonceSize := gcm.NonceSize()
		if len(ciphertext) < nonceSize {
			return nil, err
		}

		nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return nil, err
		}

		// Unmarshal proto
		result[i] = &commonpb.Payload{}
		err = result[i].Unmarshal(plaintext)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Encode implements converter.PayloadCodec.
func (c *codec) Encode(payloads []*common.Payload) ([]*common.Payload, error) {
	result := make([]*commonpb.Payload, len(payloads))
	for i, p := range payloads {
		// Marshal proto
		origBytes, err := p.Marshal()
		if err != nil {
			return payloads, err
		}

		// Use the first key to encrypt
		key := c.keys[0]

		gcm, err := c.newCipher(key)
		if err != nil {
			return nil, err
		}

		// Create a new byte array
		nonce := make([]byte, gcm.NonceSize())

		// Create a cryptographically secure random sequence
		if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
			return nil, fmt.Errorf("error reading random nonce: %w", err)
		}

		// Encrypt the data
		b := gcm.Seal(nonce, nonce, origBytes, nil)

		result[i] = &commonpb.Payload{
			Metadata: map[string][]byte{
				converter.MetadataEncoding: []byte(AESMimeType),
				MetadataKeyID:              []byte(key.ID),
			},
			Data: b,
		}
	}

	return result, nil
}

func (c *codec) newCipher(key Key) (cipher.AEAD, error) {
	// Generate a new cipher with the encryption key
	a, err := aes.NewCipher([]byte(key.Key))
	if err != nil {
		return nil, fmt.Errorf("error creating aes cipher: %w", err)
	}

	// Create a Galois counter mode for symmetric keys
	// @link https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(a)
	if err != nil {
		return nil, fmt.Errorf("error create galois counter mode: %w", err)
	}

	return gcm, nil
}

func DataConverter(keys Keys) converter.DataConverter {
	return NewDataConverter(converter.GetDefaultDataConverter(), keys)
}

func NewPayloadCodec(keys Keys) converter.PayloadCodec {
	return &codec{keys: keys}
}

// NewDataConverter creates a new data converter that wraps the converter
func NewDataConverter(underlying converter.DataConverter, keys Keys) converter.DataConverter {
	return converter.NewCodecDataConverter(underlying, NewPayloadCodec(keys))
}

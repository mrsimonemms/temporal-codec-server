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
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Keys []Key

type Key struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

func ReadKeyFile(filepath string) (Keys, error) {
	jsonFile, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer func() {
		if err := jsonFile.Close(); err != nil {
			err = fmt.Errorf("error closing file: %w", err)
		}
	}()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var keys Keys
	if err := json.Unmarshal(byteValue, &keys); err != nil {
		return nil, fmt.Errorf("error unmarshalling json: %w", err)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("at least one key is required")
	}

	return keys, err
}

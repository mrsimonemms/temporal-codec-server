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

package main

import (
	"context"
	"fmt"
	"os"

	gh "github.com/mrsimonemms/golang-helpers"
	"github.com/rs/zerolog/log"
	temporal "github.com/zigflow/helpers"
	"github.com/zigflow/zigflow/pkg/codec"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

func exec() error {
	log.Info().Msg("Triggering a hello world encryption example")

	dataConverter, err := codec.NewAESConverter(os.Getenv("CONVERTER_KEY_PATH"))
	if err != nil {
		return fmt.Errorf("error creating aes converter: %w", err)
	}

	c, err := temporal.NewConnectionWithEnvvars(
		temporal.WithDataConverter(converter.NewCodecDataConverter(dataConverter)),
		temporal.WithZerolog(&log.Logger),
	)
	if err != nil {
		return fmt.Errorf("error connecting to temporal: %w", err)
	}
	defer c.Close()

	ctx := context.TODO()

	opts := client.StartWorkflowOptions{
		TaskQueue: "zigflow",
	}

	we, err := c.ExecuteWorkflow(ctx, opts, "encryption", map[string]any{
		"name": "Example",
	})
	if err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error executing workflow",
		}
	}

	var result any
	if err := we.Get(ctx, &result); err != nil {
		return gh.FatalError{
			Cause: err,
			Msg:   "Error getting response",
		}
	}

	log.Info().Any("result ", result).Msg("Encryption example completed")

	return nil
}

func main() {
	if err := exec(); err != nil {
		os.Exit(gh.HandleFatalError(err))
	}
}

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

package e2e_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/mrsimonemms/temporal-codec-server/examples/golang"
	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/remote"
	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func runner(b *testing.B, name string, opts client.Options) {
	opts.HostPort = os.Getenv("TEMPORAL_ADDRESS")

	c, err := client.Dial(opts)
	assert.NoError(b, err)
	defer c.Close()

	// Start the worker
	w := worker.New(c, "hello-world", worker.Options{})

	w.RegisterWorkflow(golang.Workflow)
	w.RegisterActivity(golang.Activity)

	go func() {
		err = w.Run(worker.InterruptCh())
		assert.NoError(b, err)
	}()

	i := 0
	for b.Loop() {
		workflowOptions := client.StartWorkflowOptions{
			ID:        fmt.Sprintf("%s_%d", name, i),
			TaskQueue: "hello-world",
		}

		input := fmt.Sprintf("Run %s %d", name, i)

		ctx := context.Background()

		we, err := c.ExecuteWorkflow(ctx, workflowOptions, golang.Workflow, input)
		assert.NoError(b, err)

		var result string
		err = we.Get(ctx, &result)
		assert.NoError(b, err)

		assert.Equal(b, fmt.Sprintf("Hello %s!", input), result)

		i++
	}
}

func BenchmarkNoConversion(b *testing.B) {
	runner(b, "NoConversion", client.Options{})
}

func BenchmarkLocalAESConversion(b *testing.B) {
	runner(b, "AESConversion", client.Options{
		DataConverter: aes.DataConverter(aes.Keys{
			{
				ID:  "key0",
				Key: "passphrasewhichneedstobe32bytes!",
			},
			{
				ID:  "key1",
				Key: "anoldpassphraseinourhistory!!!!!",
			},
		}),
	})
}

func BenchmarkRemoteConversion(b *testing.B) {
	runner(b, "RemoteConversion", client.Options{
		DataConverter: remote.DataConverter(os.Getenv("CODEC_SERVER_ADDRESS")),
	})
}

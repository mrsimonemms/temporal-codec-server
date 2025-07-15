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
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type log struct{}

func (l log) Debug(a string, b ...any) {}
func (l log) Error(a string, b ...any) {}
func (l log) Info(a string, b ...any)  {}
func (l log) Warn(a string, b ...any)  {}

func runner(b *testing.B, name string, opts client.Options) {
	b.Helper()

	// ---- setup (not timed) ----
	b.StopTimer()

	opts.HostPort = os.Getenv("TEMPORAL_ADDRESS")
	opts.Logger = log{}

	c, err := client.Dial(opts)
	if err != nil {
		b.Fatal(err)
	}
	defer c.Close()

	// Unique task queue per benchmark name + pid to avoid queue contention
	tq := fmt.Sprintf("hello-world-%s-%d", name, os.Getpid())

	w := worker.New(c, tq, worker.Options{})
	// Register BEFORE starting the worker
	w.RegisterWorkflow(golang.Workflow)
	w.RegisterActivity(golang.Activity)

	if err := w.Start(); err != nil {
		b.Fatal(err)
	}
	defer w.Stop()

	ctx := context.Background()

	b.StartTimer()
	b.ResetTimer()

	i := 0
	for b.Loop() {
		workflowOptions := client.StartWorkflowOptions{
			ID:        fmt.Sprintf("%s_%d", name, i),
			TaskQueue: tq,
		}

		input := fmt.Sprintf("Run %s %d", name, i)
		we, err := c.ExecuteWorkflow(ctx, workflowOptions, golang.Workflow, input)
		if err != nil {
			b.Fatal(err)
		}

		var result string
		if err := we.Get(ctx, &result); err != nil {
			b.Fatal(err)
		}

		expected := fmt.Sprintf("Hello %s!", input)
		if result != expected {
			b.Fatalf("unexpected result: got %q want %q", result, expected)
		}

		i++
	}
}

// Run with no data conversion. This should be fastest
func BenchmarkNoConversion(b *testing.B) {
	b.ReportAllocs()
	runner(b, "NoConversion", client.Options{})
}

// Locally convert via AES.
func BenchmarkLocalAESConversion(b *testing.B) {
	b.ReportAllocs()
	runner(b, "AESConversion", client.Options{
		DataConverter: aes.DataConverter(aes.Keys{
			{ID: "key0", Key: "passphrasewhichneedstobe32bytes!"},
			{ID: "key1", Key: "anoldpassphraseinourhistory!!!!!"},
		}),
	})
}

// Remotely (via HTTP) convert via AES. It's the same algorithm as above, but
// web latency
func BenchmarkRemoteConversion(b *testing.B) {
	b.ReportAllocs()
	runner(b, "RemoteConversion", client.Options{
		DataConverter: remote.DataConverter(os.Getenv("CODEC_SERVER_ADDRESS")),
	})
}

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
	_ "embed"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/remote"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type log struct{}

func (l log) Debug(a string, b ...any) {}
func (l log) Error(a string, b ...any) {}
func (l log) Info(a string, b ...any)  {}
func (l log) Warn(a string, b ...any)  {}

func Workflow(ctx workflow.Context, data any) (any, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	logger := workflow.GetLogger(ctx)
	logger.Info("Workflow started")

	var result any
	err := workflow.ExecuteActivity(ctx, Activity, data).Get(ctx, &result)
	if err != nil {
		logger.Error("Activity failed.", "Error", err)
		return "", err
	}

	logger.Info("Workflow completed.")

	return result, nil
}

func Activity(ctx context.Context, name any) (any, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Activity")
	return name, nil
}

func runner(b *testing.B, name string, opts client.Options, inputData func(name string, i int) any) {
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
	tq := fmt.Sprintf("benchmark-%s-%d", name, os.Getpid())

	w := worker.New(c, tq, worker.Options{})

	w.RegisterWorkflow(Workflow)
	w.RegisterActivity(Activity)

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

		input := inputData(name, i)
		we, err := c.ExecuteWorkflow(ctx, workflowOptions, Workflow, input)
		if err != nil {
			b.Fatal(err)
		}

		var result any
		if err := we.Get(ctx, &result); err != nil {
			b.Fatal(err)
		}

		i++
	}
}

func simpleData(name string, i int) any {
	return fmt.Sprintf("Run %s %d", name, i)
}

//go:embed testdata/shakespeare.txt
var shakespeareText string

//go:embed testdata/shakespeare100kb.txt
var shakespeare100kbText string

// Get a load of Shakespeare. It's not important what it is, just
// that it's something between 1 and 1.5MB to really stretch what
// the tests are doing. The limit for Temporal is 2MB, but that's on
// the encrypted data, so we need to leave something for the encryption
// to take up in case it gets larger, which it almost certainly will.
func shakespeare(_ string, _ int) any {
	return shakespeareText
}

func shakespeare100kb(_ string, _ int) any {
	return shakespeare100kbText
}

func complexData(name string, i int) any {
	type obj struct {
		SomeString string
		SomeNumber float64
		SomeBool   bool
	}

	type data struct {
		Name       string
		ID         int
		SomeString string
		SomeNumber float64
		SomeBool   bool
		SomeObj    obj
		SomeArray  []obj
	}

	return data{
		Name: name,
		ID:   i,
		SomeString: `Hello. This is a delightful string. Not too long, but does the job

Let's put a line break in. Why? Whyever not?`,
		SomeNumber: 98103984093891093,
		SomeBool:   true,
		SomeObj: obj{
			SomeString: "This is a much shorter string. Harrumble! Hello " + name,
			SomeNumber: 3.141,
			SomeBool:   false,
		},
		SomeArray: []obj{
			{
				SomeString: "String 1",
				SomeNumber: 12345,
				SomeBool:   true,
			},
			{
				SomeString: "String 2",
				SomeNumber: 2468,
				SomeBool:   false,
			},
		},
	}
}

// Run with no data conversion. This should be fastest
func BenchmarkNoConversion(b *testing.B) {
	b.ReportAllocs()
	runner(b, "NoConversion", client.Options{}, simpleData)
}

func BenchmarkNoConversionComplexData(b *testing.B) {
	b.ReportAllocs()
	runner(b, "NoConversionComplexData", client.Options{}, complexData)
}

func BenchmarkNoConversionShakespeare100kb(b *testing.B) {
	b.ReportAllocs()
	runner(b, "NoConversionShakespeare100kb", client.Options{}, shakespeare100kb)
}

func BenchmarkNoConversionShakespeare(b *testing.B) {
	b.ReportAllocs()
	runner(b, "NoConversionShakespeare", client.Options{}, shakespeare)
}

// Locally convert via AES.
func BenchmarkLocalAESConversion(b *testing.B) {
	b.ReportAllocs()
	runner(b, "AESConversion", client.Options{
		DataConverter: aes.DataConverter(aes.Keys{
			{ID: "key0", Key: "passphrasewhichneedstobe32bytes!"},
			{ID: "key1", Key: "anoldpassphraseinourhistory!!!!!"},
		}),
	}, simpleData)
}

func BenchmarkLocalAESConversionComplexData(b *testing.B) {
	b.ReportAllocs()
	runner(b, "AESConversionComplexData", client.Options{
		DataConverter: aes.DataConverter(aes.Keys{
			{ID: "key0", Key: "passphrasewhichneedstobe32bytes!"},
			{ID: "key1", Key: "anoldpassphraseinourhistory!!!!!"},
		}),
	}, complexData)
}

func BenchmarkLocalAESConversionShakespeare100kb(b *testing.B) {
	b.ReportAllocs()
	runner(b, "AESConversionShakespeare100kb", client.Options{
		DataConverter: aes.DataConverter(aes.Keys{
			{ID: "key0", Key: "passphrasewhichneedstobe32bytes!"},
			{ID: "key1", Key: "anoldpassphraseinourhistory!!!!!"},
		}),
	}, shakespeare100kb)
}

func BenchmarkLocalAESConversionShakespeare(b *testing.B) {
	b.ReportAllocs()
	runner(b, "AESConversionShakespeare", client.Options{
		DataConverter: aes.DataConverter(aes.Keys{
			{ID: "key0", Key: "passphrasewhichneedstobe32bytes!"},
			{ID: "key1", Key: "anoldpassphraseinourhistory!!!!!"},
		}),
	}, shakespeare)
}

// Remotely (via HTTP) convert via AES. It's the same algorithm as above, but
// web latency
func BenchmarkRemoteConversion(b *testing.B) {
	b.ReportAllocs()
	runner(b, "RemoteConversion", client.Options{
		DataConverter: remote.DataConverter(os.Getenv("CODEC_SERVER_ADDRESS")),
	}, simpleData)
}

func BenchmarkRemoteConversionComplexData(b *testing.B) {
	b.ReportAllocs()
	runner(b, "RemoteConversionComplexData", client.Options{
		DataConverter: remote.DataConverter(os.Getenv("CODEC_SERVER_ADDRESS")),
	}, complexData)
}

func BenchmarkRemoteConversionShakespeare100kb(b *testing.B) {
	b.ReportAllocs()
	runner(b, "RemoteConversionShakespeare100kb", client.Options{
		DataConverter: remote.DataConverter(os.Getenv("CODEC_SERVER_ADDRESS")),
	}, shakespeare100kb)
}

func BenchmarkRemoteConversionShakespeare(b *testing.B) {
	b.ReportAllocs()
	runner(b, "RemoteConversionShakespeare", client.Options{
		DataConverter: remote.DataConverter(os.Getenv("CODEC_SERVER_ADDRESS")),
	}, shakespeare)
}

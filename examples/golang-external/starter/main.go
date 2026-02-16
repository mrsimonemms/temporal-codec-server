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
	"log"
	"os"

	"golang"

	"github.com/mrsimonemms/golang-helpers/temporal"
	"github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/external"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/client"
)

func main() {
	// Connect to Redis
	redisAddress := "localhost:6379"
	if r, ok := os.LookupEnv("REDIS_ADDRESS"); ok {
		redisAddress = r
	}
	connection, err := external.NewRedis(context.Background(), &redis.Options{
		Addr: redisAddress,
	})
	if err != nil {
		log.Fatalln("Unable to get keys from file", err)
	}
	defer func() {
		if err := connection.Close(); err != nil {
			log.Println("Error closing Redis connect", err)
		}
	}()

	// The client and worker are heavyweight objects that should be created once per process.
	c, err := temporal.NewConnectionWithEnvvars(
		temporal.WithDataConverter(external.DataConverter(connection)),
	)
	if err != nil {
		log.Fatalln("Unable to create client", err)
	}
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{
		ID:        "hello_world_workflowID",
		TaskQueue: "hello-world",
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, golang.Workflow, "Temporal")
	if err != nil {
		log.Fatalln("Unable to execute workflow", err)
	}

	log.Println("Started workflow", "WorkflowID", we.GetID(), "RunID", we.GetRunID())

	// Synchronously wait for the workflow completion.
	var result string
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalln("Unable get workflow result", err)
	}
	log.Println("Workflow result:", result)
}

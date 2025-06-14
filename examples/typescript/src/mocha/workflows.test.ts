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

// @@@SNIPSTART hello-world-project-template-ts-workflow-test
import { TestWorkflowEnvironment } from '@temporalio/testing';
import { before, describe, it } from 'mocha';
import { Worker } from '@temporalio/worker';
import { example } from '../workflows';
import * as activities from '../activities';
import assert from 'assert';

describe('Example workflow', () => {
  let testEnv: TestWorkflowEnvironment;

  before(async () => {
    testEnv = await TestWorkflowEnvironment.createLocal();
  });

  after(async () => {
    await testEnv?.teardown();
  });

  it('successfully completes the Workflow', async () => {
    const { client, nativeConnection } = testEnv;
    const taskQueue = 'test';

    const worker = await Worker.create({
      connection: nativeConnection,
      taskQueue,
      workflowsPath: require.resolve('../workflows'),
      activities,
    });

    const result = await worker.runUntil(
      client.workflow.execute(example, {
        args: ['Temporal'],
        workflowId: 'test',
        taskQueue,
      }),
    );
    assert.equal(result, 'Hello, Temporal!');
  });
});
// @@@SNIPEND

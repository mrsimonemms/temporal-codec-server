# temporal-codec-server

Encode and decode your Temporal data with TypeScript

<!-- toc -->

* [Temporal SDK example](#temporal-sdk-example)
  * [Installation](#installation)
  * [Client](#client)
  * [Worker](#worker)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Temporal SDK example

> See [keys.example.yaml](https://github.com/mrsimonemms/temporal-codec-server/blob/e11e08a51b0cc0673363e6df3d4d4280319bce2b/keys.example.yaml)
> for an example key file.
>
> For best results, use an environment variable rather than hardcoding the file
> path.

### Installation

```sh
npm install --save @mrsimonemms/temporal-codec-server
```

### Client

```typescript
import { Connection, Client } from '@temporalio/client';
import { AESCodec } from '@mrsimonemms/temporal-codec-server';

async function run() {
  const connection = await Connection.connect();

  const client = new Client({
    connection,
    dataConverter: {
      // Load the payload converter
      payloadCodecs: [await AESCodec.create('/path/to/keyfile')],
    },
    // For additional options, see https://docs.temporal.io/develop/typescript/temporal-clients#connect-to-development-service
  });

  // Continue...
}

run().catch((err) => {
  console.error(err);
  process.exit(1);
});
```

### Worker

```typescript
import { NativeConnection, Worker } from '@temporalio/worker';
import { AESCodec } from '@mrsimonemms/temporal-codec-server';

async function run() {
  const connection = await NativeConnection.connect();

  try {
    const worker = await Worker.create({
      dataConverter: {
        // Load the payload converter
        payloadCodecs: [await AESCodec.create('/path/to/keyfile')],
      },
      connection,
    // For additional options, see https://docs.temporal.io/develop/typescript/temporal-clients#set-task-queue
    });

    // Continue...
  } catch {
    await connection.close();
  }
}

run().catch((err) => {
  console.error(err);
  process.exit(1);
});
```

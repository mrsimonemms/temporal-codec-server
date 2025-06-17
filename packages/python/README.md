# temporal-codec

Encode and decode your Temporal data with Python

<!-- toc -->

* [Temporal SDK example](#temporal-sdk-example)
  * [Installation](#installation)
  * [Example](#example)

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
pip install temporalcodec
```

### Example

```python
from dataclasses import dataclass_replace
from temporalio.client import Client
import temporalio.converter

from temporalcodec.encryption import EncryptionCodec


async def main():
    client = await Client.connect(
        "localhost:7233",
        data_converter=dataclass_replace(
            # Create the EncryptionCodec with keys loaded to memory
            temporalio.converter.default(), payload_codec=await EncryptionCodec.create(keypath="/path/to/keyfile")
        ),
    )

if __name__ == "__main__":
    asyncio.run(main())
```

# temporal-codec-server

Encode and decode your Temporal data with Golang

<!-- toc -->

* [Temporal SDK example](#temporal-sdk-example)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Temporal SDK example

> See [keys.example.yaml](https://github.com/mrsimonemms/temporal-codec-server/blob/e11e08a51b0cc0673363e6df3d4d4280319bce2b/keys.example.yaml)
> for an example key file.
>
> For best results, use an environment variable rather than hardcoding the file
> path.

```sh
go get github.com/mrsimonemms/temporal-codec-server/packages/golang
```

```go
package main

import (
  "github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
  "go.temporal.io/sdk/client"
)

func main() {
  // Load the encryption keys to memory
  keys, err := aes.ReadKeyFile("/path/to/keyfile")
  if err != nil {
    panic(err)
  }

  c, err := client.Dial(client.Options{
    DataConverter: aes.DataConverter(keys),
  })
  if err != nil {
    panic(err)
  }
  defer c.Close()

  // Continue...
}
```

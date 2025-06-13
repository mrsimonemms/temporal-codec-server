# Go

Golang implementation

<!-- toc -->

* [Docker Image](#docker-image)
* [SDK usage](#sdk-usage)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Docker Image

`ghcr.io/mrsimonemms/temporal-codec-server/golang`

## SDK usage

```go
package main

import (
  "log"
  "os"

  "github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/aes"
  "go.temporal.io/sdk/client"
)

func main() {
  // Load the encryption keys
  keys, err := aes.ReadKeyFile(os.Getenv("KEYS_PATH"))
  if err != nil {
    log.Fatalln("Unable to get keys from file", err)
  }

  // Create your Temporal client
  c, err := client.Dial(client.Options{
    // Load the AES dataconverter
    DataConverter: aes.DataConverter(keys),
  })
  if err != nil {
    log.Fatalln("Unable to create client", err)
  }
  defer c.Close()

  // Continue with your Temporal application
}
```

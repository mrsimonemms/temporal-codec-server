# Remote

A very silly idea to use a Codec Server to encode and decode your payloads

<!-- toc -->

* [Why is this a silly idea?](#why-is-this-a-silly-idea)
* [Why did you build it then?](#why-did-you-build-it-then)
* [Ok, how do I use it?](#ok-how-do-i-use-it)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Why is this a silly idea?

This will be a huge bottleneck. Temporal is designed to be fast. Encoding,
especially AES, is an expensive process by design. This will probably make your
Temporal calls slow and potentially unreliable. Temporal is a highly scalable
service.

This Codec Server is not.

Also, the `POST:/encode` endpoints are, by design, unauthenticated endpoints. So
this can be open to anyone who wants to use it. They also won't be able to decode
anything that comes out of it.

## Why did you build it then?

I wanted to see just how bad an idea it is. All my brilliant Solutions Architect
colleagues say we shouldn't do remote data conversion. And they understand why
because they've experienced people who have done it.

But I haven't done. By seeing how silly an idea this is why allow me to better
advise the people I work with and advise.

I'm the sort of person who, when told by waiting staff "be careful, that plate
is hot" checks just how hot it is. 🔥

Simon Emms - taking pain in the name of empiricism since 1991.

## Ok, how do I use it?

```sh
go get github.com/mrsimonemms/temporal-codec-server/packages/golang
```

```go
package main

import (
  "github.com/mrsimonemms/temporal-codec-server/packages/golang/algorithms/remote"
  "go.temporal.io/sdk/client"
)

func main() {
  // The client is a heavyweight object that should be created once per process.
  c, err := client.Dial(client.Options{
    DataConverter: remote.DataConverter("http://localhost:3000"), // Replace with the URL you want to use
  })
  if err != nil {
    panic(err)
  }
  defer c.Close()

  // Continue...
```

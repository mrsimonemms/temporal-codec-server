# temporal-codec-server

Encode and decode your Temporal data

<!-- toc -->

* [Purpose](#purpose)
  * [Key file format](#key-file-format)
  * [Languages](#languages)
* [Deployment](#deployment)
* [Contributing](#contributing)
  * [Open in a container](#open-in-a-container)
  * [Commit style](#commit-style)

<!-- Regenerate with "pre-commit run -a markdown-toc" -->

<!-- tocstop -->

## Purpose

This repository is designed to act as a guide for creating a
[Codec Server](https://docs.temporal.io/production-deployment/data-encryption)
for every language [officially supported by Temporal](https://docs.temporal.io/encyclopedia/temporal-sdks#official-sdks).

For every language, these implement an [AES](https://en.wikipedia.org/wiki/Advanced_Encryption_Standard)
encryption algorithm and add an `encryption-key-id` to the metadata. Multiple
keys can be added to allow for key rotation.

### Key file format

Both YAML and JSON are supported for the key file. It's a simple structure where
the active key is the key in position 0.

The `id` is stored in the metadata and will be publicly visible so should not be
considered sensitive data. The `key` is used by the algorithm and should be
treated as a secret.

```yaml
- id: key0
  key: passphrasewhichneedstobe32bytes!
- id: key1
  key: anoldpassphraseinourhistory!!!!!
```

### Languages

* [Go](./apps/golang)
* Java - coming soon
* PHP - coming soon
* Python - coming soon
* TypeScript - coming soon
* .Net - coming soon
* Ruby - coming soon

## Deployment

See the [Helm chart](./charts/temporal-codec-server).

## Contributing

### Open in a container

* [Open in a container](https://code.visualstudio.com/docs/devcontainers/containers)

### Commit style

All commits must be done in the [Conventional Commit](https://www.conventionalcommits.org)
format.

```git
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

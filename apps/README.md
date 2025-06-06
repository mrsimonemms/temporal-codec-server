# Apps Directory

A standalone app that runs in a standalone fashion, typically in a container.

These could be user-facing apps, such as a website or API, or a microservice
that provides a backing service.

These apps require either:
* a `Dockerfile` in their root, or
* a `.build.yaml` in their root referencing a common Dockerfile

## Examples .build.yaml

```yaml
dockerfile: ../golang.Dockerfile # File paths are relative
```

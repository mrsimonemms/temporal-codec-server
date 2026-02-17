# Golang External

Follow [Hello World](https://github.com/temporalio/samples-go/tree/main/helloworld)
instructions

## Running Examples

You will need to set the `CONNECTION_TYPE` environment variable. Supported
connections are:

* `redis`
* `s3`

The Temporal UI will be on [localhost:8080](http://localhost:8080).

### Running the worker

```sh
docker compose up worker-${CONNECTION_TYPE} --build --watch
```

### Running the starter

```sh
docker compose up starter-${CONNECTION_TYPE} --build
```

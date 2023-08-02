# Nexus API

Protocol Buffers API definitions for Nexus RPCs.

### Prerequisites

- [go](https://go.dev/doc/install)
- [protoc](https://grpc.io/docs/protoc-installation/)

### Install lint and test tool dependencies

```shell
go run ./cmd install-deps
```

### Lint

Lint with:

- [buf](https://buf.build/docs/installation)
- [api-linter](https://github.com/googleapis/api-linter)

```shell
go run ./cmd lint
```

### Test compilation

Test protoc compilation for select languages.

```shell
go run ./cmd test
```

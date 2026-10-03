# kubecart

A mini checkout system built as a learning project for system design and infrastructure.

```
client --HTTP--> gateway --gRPC--> order-service --gRPC--> inventory-service
```

The goal is to understand the request flow and how to operate it, not only to have working code.
The project is built in phases, from plain gRPC locally up to Kubernetes, the Gateway API and observability.

## Status

**Phase 0 (scaffold) is done.** The repo skeleton, protobuf tooling and Makefile are in place.
No business logic yet. See [LEARNINGS.md](LEARNINGS.md) for notes per phase.

| Phase | Topic                                         | Status  |
|-------|-----------------------------------------------|---------|
| 0     | Scaffold: monorepo, buf, Makefile             | done    |
| 1     | gRPC locally, in-memory storage               | planned |
| 2     | Docker, Compose, Postgres                     | planned |
| 2b    | Nginx as reverse proxy                        | planned |
| 3     | Kubernetes (kind)                             | planned |
| 4     | Gateway API                                   | planned |
| 5     | Scaling and gRPC load balancing               | planned |
| 6     | Observability (OpenTelemetry + Grafana stack) | planned |
| 7     | Chaos experiments                             | planned |

## Tech stack

- Go for all services, gRPC between services, REST at the gateway
- PostgreSQL with pgx (no ORM), golang-migrate, one database per service
- Docker, Kubernetes (kind), Nginx, Kubernetes Gateway API
- OpenTelemetry, Grafana, Tempo, Loki

## Layout

```
cmd/
  gateway/            HTTP entrypoint (main only, wiring)
  order-service/      gRPC OrderService
  inventory-service/  gRPC InventoryService
proto/                .proto contracts (order/v1, inventory/v1)
gen/go/               generated Go code (committed)
buf.yaml              buf module + lint config
buf.gen.yaml          code generation config
Makefile              common tasks
```

## Prerequisites

- Go 1.27.1. An older local Go downloads it automatically via `GOTOOLCHAIN=auto`.
- Nothing else to install: buf, protoc-gen-go and protoc-gen-go-grpc are pinned as
  `tool` dependencies in `go.mod` and run with `go tool <name>`.

## Usage

```sh
make proto          # lint .proto files and regenerate gen/go
make build          # build all services into ./bin
make test           # run tests with the race detector
make run-gateway
make run-order
make run-inventory
make clean
```

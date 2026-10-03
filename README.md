# kubecart

A mini checkout system built as a learning project for system design and infrastructure.

```
client --HTTP--> gateway --gRPC--> order-service --gRPC--> inventory-service
```

The goal is to understand the request flow and how to operate it, not only to have working code.
The project is built in phases, from plain gRPC locally up to Kubernetes, the Gateway API and observability.

## Status

**Phase 1 is done.** The three services talk gRPC locally with in-memory storage (no database yet).
See [LEARNINGS.md](LEARNINGS.md) for notes per phase.

| Phase | Topic                                         | Status  |
|-------|-----------------------------------------------|---------|
| 0     | Scaffold: monorepo, buf, Makefile             | done    |
| 1     | gRPC locally, in-memory storage               | done    |
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
internal/
  gateway/            REST handler, gRPC code -> HTTP status mapping
  order/              OrderService server + in-memory store
  inventory/          InventoryService server + in-memory store
  grpcx/              shared logging interceptor and graceful shutdown
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

## Try it

Run each service in its own terminal: `make run-inventory`, `make run-order`, `make run-gateway`.

```sh
curl -X POST localhost:8080/orders -d '{"sku":"apple","quantity":3}'   # 201
curl -X POST localhost:8080/orders -d '{"sku":"durian","quantity":1}'  # 404 unknown sku
curl -X POST localhost:8080/orders -d '{"sku":"cherry","quantity":1}'  # 409 out of stock

# Call inventory directly over gRPC
go tool buf curl --schema proto --protocol grpc --http2-prior-knowledge \
  -d '{"sku":"apple"}' http://localhost:50052/inventory.v1.InventoryService/CheckStock
```

Seed stock: apple 10, banana 5, cherry 0. Ports: gateway 8080, order 50051, inventory 50052
(override with `HTTP_ADDR`, `GRPC_ADDR`, `ORDER_ADDR`, `INVENTORY_ADDR`).

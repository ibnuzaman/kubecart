# kubecart

A mini checkout system built as a learning project for system design and infrastructure.

```
client --HTTP--> gateway --gRPC--> order-service --gRPC--> inventory-service
```

The goal is to understand the request flow and how to operate it, not only to have working code.
The project is built in phases, from plain gRPC locally up to Kubernetes, the Gateway API and observability.

## Status

**Phase 2 is done.** The services run in Docker Compose, backed by one Postgres instance with a
separate database per service.
See [LEARNINGS.md](LEARNINGS.md) for notes per phase.

| Phase | Topic                                         | Status  |
|-------|-----------------------------------------------|---------|
| 0     | Scaffold: monorepo, buf, Makefile             | done    |
| 1     | gRPC locally, in-memory storage               | done    |
| 2     | Docker, Compose, Postgres                     | done    |
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
  order/              OrderService server, Postgres store, migrations/
  inventory/          InventoryService server, Postgres store, migrations/
  grpcx/              shared logging interceptor, health, graceful shutdown
  db/                 shared Postgres connect-with-retry and migration runner
deploy/postgres/init/ creates one database + role per service (first start only)
Dockerfile            one multi-stage Dockerfile, --build-arg SERVICE=<cmd dir>
docker-compose.yml    Postgres, migrate jobs, the three services
proto/                .proto contracts (order/v1, inventory/v1)
gen/go/               generated Go code (committed)
buf.yaml              buf module + lint config
buf.gen.yaml          code generation config
Makefile              common tasks
```

## Prerequisites

- Go 1.27.1. An older local Go downloads it automatically via `GOTOOLCHAIN=auto`.
- Docker with Compose v2+ (tested with Docker 29.1, Compose v5.0).
- Nothing else to install: buf, protoc-gen-go and protoc-gen-go-grpc are pinned as
  `tool` dependencies in `go.mod` and run with `go tool <name>`.

## Usage

```sh
make proto          # lint .proto files and regenerate gen/go
make build          # build all services into ./bin
make test           # run tests with the race detector
make up             # build images, start the stack, wait until healthy
make logs
make down           # stop (data kept); `docker compose down -v` also wipes the database
make clean
```

## Try it

```sh
make up
curl -X POST localhost:8080/orders -d '{"sku":"apple","quantity":3}'   # 201
curl -X POST localhost:8080/orders -d '{"sku":"durian","quantity":1}'  # 404 unknown sku
curl -X POST localhost:8080/orders -d '{"sku":"cherry","quantity":1}'  # 409 out of stock

# Look inside the databases (Postgres has no published port)
docker compose exec postgres psql -U inventory -d inventory -c 'SELECT * FROM stock'
docker compose exec postgres psql -U orders -d orders -c 'SELECT * FROM orders'
```

Seed stock: apple 10, banana 5, cherry 0. Only the gateway is published, on `127.0.0.1:8080`.
Each binary also has `migrate` and `healthcheck` subcommands (see `docker-compose.yml`).

SERVICES := gateway order-service inventory-service
BIN_DIR  := bin

.PHONY: proto build test run-gateway run-order run-inventory certs up down logs clean

## proto: lint .proto files and regenerate Go code into gen/go
proto:
	go tool buf lint
	go tool buf generate

## build: compile every service into ./bin
build:
	@mkdir -p $(BIN_DIR)
	@for s in $(SERVICES); do \
		echo "building $$s"; \
		go build -o $(BIN_DIR)/$$s ./cmd/$$s || exit 1; \
	done

## test: run all tests with the race detector
test:
	go test -race ./...

## run-*: run one service on the host. order and inventory need DATABASE_URL
## pointing at a reachable Postgres; for the full stack use `make up`.
run-gateway:
	go run ./cmd/gateway

run-order:
	go run ./cmd/order-service

run-inventory:
	go run ./cmd/inventory-service

## certs: self-signed TLS cert for nginx (localhost), created once
CERT_DIR := deploy/nginx/certs
certs: $(CERT_DIR)/localhost.crt

$(CERT_DIR)/localhost.crt:
	@mkdir -p $(CERT_DIR)
	openssl req -x509 -newkey rsa:2048 -noenc -days 365 \
		-subj "/CN=localhost" \
		-addext "subjectAltName=DNS:localhost,IP:127.0.0.1" \
		-keyout $(CERT_DIR)/localhost.key -out $@

## up: build images and start the whole stack, waiting until it is healthy
up: certs
	docker compose up --build -d --wait

## down: stop the stack (data stays in the pgdata volume; add -v to wipe it)
down:
	docker compose down

logs:
	docker compose logs -f

clean:
	rm -rf $(BIN_DIR)

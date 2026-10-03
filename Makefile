SERVICES := gateway order-service inventory-service
BIN_DIR  := bin

.PHONY: proto build test run-gateway run-order run-inventory clean

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

run-gateway:
	go run ./cmd/gateway

run-order:
	go run ./cmd/order-service

run-inventory:
	go run ./cmd/inventory-service

clean:
	rm -rf $(BIN_DIR)

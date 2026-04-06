BINARY_NAME   := relayer
BINARY_OUTPUT := bin/$(BINARY_NAME)
ABI_FILE      := bindings/abi/VynxSettlement.json
BINDINGS_OUT  := bindings/vynx_settlement.go
BINDINGS_PKG  := bindings
BINDINGS_TYPE := VynxSettlement
GOPATH_BIN    := $(shell go env GOPATH)/bin

.PHONY: all build test lint bindings tidy clean

all: tidy bindings build

## build: Compile the relayer binary to bin/relayer
build:
	go build -o $(BINARY_OUTPUT) ./cmd/relayer/...

## test: Run the full test suite with the race detector enabled
test:
	go test -race -v -count=1 ./...

## lint: Run golangci-lint with the project configuration
lint:
	golangci-lint run ./...

## bindings: Generate Go ABI bindings from bindings/abi/VynxSettlement.json
bindings:
	@mkdir -p bindings
	$(GOPATH_BIN)/abigen \
		--abi $(ABI_FILE) \
		--pkg $(BINDINGS_PKG) \
		--type $(BINDINGS_TYPE) \
		--out $(BINDINGS_OUT)

## tidy: Synchronise go.mod and go.sum
tidy:
	go mod tidy

## simulate: Run the live E2E simulation against a running Anvil node and relayer
simulate:
	go run ./cmd/simulate/...

## clean: Remove compiled binaries
clean:
	rm -rf bin/

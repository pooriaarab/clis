.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -o bin/clarity-pp-cli$(BIN_EXT) ./cmd/clarity-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/clarity-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/clarity-pp-mcp$(BIN_EXT) ./cmd/clarity-pp-mcp

install-mcp:
	go install ./cmd/clarity-pp-mcp

build-all: build build-mcp

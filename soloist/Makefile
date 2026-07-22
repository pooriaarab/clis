.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -o bin/soloist-pp-cli$(BIN_EXT) ./cmd/soloist-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/soloist-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/soloist-pp-mcp$(BIN_EXT) ./cmd/soloist-pp-mcp

install-mcp:
	go install ./cmd/soloist-pp-mcp

build-all: build build-mcp

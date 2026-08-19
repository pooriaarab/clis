.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -o bin/cineplex-pp-cli$(BIN_EXT) ./cmd/cineplex-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/cineplex-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/cineplex-pp-mcp$(BIN_EXT) ./cmd/cineplex-pp-mcp

install-mcp:
	go install ./cmd/cineplex-pp-mcp

build-all: build build-mcp

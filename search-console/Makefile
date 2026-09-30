BINARY=bin/search-console

.PHONY: build e2e
build:
	go build -o $(BINARY) ./cmd/search-console

# The E2E tests build and run the real binary against local fake servers.
e2e:
	go test -count=1 ./e2e/...

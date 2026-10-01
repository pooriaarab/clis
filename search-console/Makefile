# macOS ships make 3.81, which ignores .SHELLFLAGS. The recipe sets pipefail itself,
# so a failing `go test` fails the target on every make.
SHELL := /bin/bash
BINARY=bin/search-console
# Drop the timings, so two runs give the same report.
STRIP=sed -E 's/ \([0-9.]+s\)//; s/^(ok|FAIL)[[:space:]]+[^[:space:]]+[[:space:]]+[0-9.]+s$$/\1 e2e/'

.PHONY: build e2e
build:
	go build -o $(BINARY) ./cmd/search-console

# The E2E tests build and run the real binary against local fake servers.
# This writes e2e-report.txt: every test result, then the transcript of one launch and its rerun.
e2e:
	go vet ./...
	set -eo pipefail; trap 'rm -f e2e-report.txt.tmp' EXIT; \
	{ echo "# Every E2E test"; \
	  go test -race -count=1 -v ./e2e/ | grep -E '^(--- |ok|FAIL)' | $(STRIP); \
	  echo; echo "# One launch, then a rerun that changes nothing"; \
	  go test -race -count=1 -v -run TestLaunchPassesThenRerunsWithoutRepeating ./e2e/ | grep -vE '^(=== |ok|PASS|--- )' | $(STRIP); \
	} > e2e-report.txt.tmp; \
	mv e2e-report.txt.tmp e2e-report.txt

SHELL := /bin/bash
.SHELLFLAGS := -eo pipefail -c
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
	{ echo "# Every E2E test"; \
	  go test -count=1 -v ./e2e/ | grep -E '^(--- |ok|FAIL)' | $(STRIP); \
	  echo; echo "# One launch, then a rerun that changes nothing"; \
	  go test -count=1 -v -run TestLaunchPassesThenRerunsWithoutRepeating ./e2e/ | grep -vE '^(=== |ok|PASS|--- )' | $(STRIP); \
	} > e2e-report.txt

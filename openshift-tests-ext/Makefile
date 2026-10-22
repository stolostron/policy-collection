BINARY := acm-policy-tests
GO_PKG_NAME := github.com/stolostron/policy-collection/openshift-tests-ext

GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_TREE_STATE := $(shell if git diff --quiet 2>/dev/null; then echo clean; else echo dirty; fi)

.PHONY: build test validate clean

build:
	go build -o $(BINARY) ./cmd/acm-policy-tests/...

test: build
	./$(BINARY) run-suite acm/policy/all

validate: build
	@echo "==> Validating extension info..."
	./$(BINARY) info
	@echo "==> Listing tests..."
	./$(BINARY) list tests --output names
	@echo "==> Listing suites..."
	./$(BINARY) list suites
	@echo "==> Validation passed."

clean:
	rm -f $(BINARY)

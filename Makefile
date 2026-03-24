GO ?= go
BIN := workflow-lock
GOFUMPT_PACKAGE ?= mvdan.cc/gofumpt@v0.9.2
GOLANGCI_LINT_PACKAGE ?= github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0

.PHONY: build dist clean test fmt lint verify lock ci

build:
	mkdir -p bin
	$(GO) build -o bin/$(BIN) ./cmd/workflow-lock

dist:
	rm -rf dist
	mkdir -p dist
	GOOS=darwin GOARCH=amd64 $(GO) build -o dist/$(BIN)_darwin_amd64 ./cmd/workflow-lock
	GOOS=darwin GOARCH=arm64 $(GO) build -o dist/$(BIN)_darwin_arm64 ./cmd/workflow-lock
	GOOS=linux GOARCH=amd64 $(GO) build -o dist/$(BIN)_linux_amd64 ./cmd/workflow-lock
	GOOS=linux GOARCH=arm64 $(GO) build -o dist/$(BIN)_linux_arm64 ./cmd/workflow-lock
	GOOS=windows GOARCH=amd64 $(GO) build -o dist/$(BIN)_windows_amd64.exe ./cmd/workflow-lock
	shasum -a 256 dist/* > dist/SHA256SUMS

clean:
	rm -rf bin dist

test:
	$(GO) test ./...

fmt:
	$(GO) run $(GOFUMPT_PACKAGE) -extra -w ./cmd ./internal

lint:
	test -z "$$($(GO) run $(GOFUMPT_PACKAGE) -extra -l ./cmd ./internal)"
	$(GO) run $(GOLANGCI_LINT_PACKAGE) run

verify:
	$(GO) run ./cmd/workflow-lock verify

lock:
	$(GO) run ./cmd/workflow-lock lock

ci:
	$(MAKE) lint
	$(GO) test ./...
	$(GO) run ./cmd/workflow-lock verify

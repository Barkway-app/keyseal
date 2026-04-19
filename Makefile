APP := keyseal
BIN := ./bin/$(APP)
PKG := github.com/Barkway-app/keyseal/internal/cli
VERSION ?= dev
LDFLAGS := -X $(PKG).Version=$(VERSION)
GO ?= go
export GOCACHE ?= $(CURDIR)/.cache/go-build

.PHONY: build test fmt fmt-check check run tidy

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/keyseal

test:
	$(GO) test ./...

fmt:
	gofmt -w cmd internal

fmt-check:
	files="$$(find cmd internal -name '*.go' -print)"; \
	test -n "$$files"; \
	test -z "$$(gofmt -l $$files)"

check: fmt-check test build

run:
	$(GO) run ./cmd/keyseal

tidy:
	$(GO) mod tidy

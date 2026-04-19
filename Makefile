APP := keyseal
BIN := ./bin/$(APP)
BUILDINFO_PKG := github.com/Barkway-app/keyseal/internal/buildinfo
VERSION ?= $(shell git describe --tags --abbrev=0 --match 'v*' 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo unknown)
LDFLAGS := -X $(BUILDINFO_PKG).Version=$(VERSION) -X $(BUILDINFO_PKG).Commit=$(COMMIT) -X $(BUILDINFO_PKG).Date=$(DATE)
GO ?= go
export GOCACHE ?= $(CURDIR)/.cache/go-build
DIST_DIR := ./dist
DIST_PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

.PHONY: build test fmt fmt-check check run tidy dist

build:
	mkdir -p ./bin
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

dist:
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)
	set -eu; \
	for platform in $(DIST_PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		archive="$(APP)_$(VERSION)_$${os}_$${arch}.tar.gz"; \
		stage="$(DIST_DIR)/$(APP)_$(VERSION)_$${os}_$${arch}"; \
		stage_name=$$(basename "$$stage"); \
		mkdir -p "$$stage"; \
		GOOS=$$os GOARCH=$$arch $(GO) build -ldflags "$(LDFLAGS)" -o "$$stage/$(APP)" ./cmd/keyseal; \
		cp README.md LICENSE "$$stage/"; \
		tar -C "$(DIST_DIR)" -czf "$(DIST_DIR)/$$archive" "$$stage_name"; \
		rm -rf "$$stage"; \
	done; \
	cd "$(DIST_DIR)"; \
	if command -v sha256sum >/dev/null 2>&1; then \
		sha256sum ./*.tar.gz > "$(APP)_$(VERSION)_checksums.txt"; \
	else \
		shasum -a 256 ./*.tar.gz > "$(APP)_$(VERSION)_checksums.txt"; \
	fi

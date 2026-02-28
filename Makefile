# GPU Economic Telemetry Agent — Makefile
# Targets: format, build, test, lint, clean, and run targets for agent, demo, and scripts.

.PHONY: format format-check build build-agent build-script build-demo test lint tidy clean run-agent run-demo run-snapshot run-frontend frontend-install frontend-build all help

# Default target
all: format build test

# Go and tools
GO       := go
GOFLAGS  :=
BINARY   := gpu-econ-agent
SCRIPT   := extract_gpu_snapshot
INGEST   := ingestion
VERSION  ?= dev

# Paths
CMD_AGENT  := ./cmd/agent
CMD_SCRIPT := ./scripts/extract_gpu_snapshot
CMD_DEMO   := ./demo/ingestion
CONFIG     ?= config.yaml

# ------------------------------------------------------------------------------
# Code formatting
# ------------------------------------------------------------------------------
# Format runs gofmt and goimports (if available) on all Go files.
format:
	$(GO) fmt ./...
	@which goimports >/dev/null 2>&1 && goimports -w -local github.com/mavvrik/gpu-economics-agent . || true

# format-check exits with failure if any Go file is not formatted (for CI).
format-check:
	@out=$$(gofmt -l .); test -z "$$out" || (echo "Unformatted Go files:"; echo "$$out"; echo "run 'make format'"; exit 1)

# ------------------------------------------------------------------------------
# Build
# ------------------------------------------------------------------------------
build: build-agent build-script build-demo

build-agent:
	$(GO) build $(GOFLAGS) -ldflags "-s -w -X main.Version=$(VERSION)" -o $(BINARY) $(CMD_AGENT)

build-script:
	$(GO) build $(GOFLAGS) -o $(SCRIPT) $(CMD_SCRIPT)

build-demo:
	$(GO) build $(GOFLAGS) -o $(INGEST) $(CMD_DEMO)

# ------------------------------------------------------------------------------
# Test and lint
# ------------------------------------------------------------------------------
test:
	$(GO) test ./...

lint:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

# ------------------------------------------------------------------------------
# Clean
# ------------------------------------------------------------------------------
clean:
	rm -f $(BINARY) $(SCRIPT) $(INGEST)
	rm -f demo/ingestion/$(INGEST)
	$(GO) clean -testcache

# ------------------------------------------------------------------------------
# Run targets (for local dev)
# ------------------------------------------------------------------------------
run-agent: build-agent
	./$(BINARY) -config $(CONFIG)

run-demo: build-demo
	./$(INGEST) -addr :4317 -out otel_ingestion.jsonl

run-snapshot: build-script
	./$(SCRIPT) -out gpu_snapshot.json

# ------------------------------------------------------------------------------
# Demo frontend (React)
# ------------------------------------------------------------------------------
frontend-install:
	cd demo/frontend && npm install

frontend-build:
	cd demo/frontend && npm run build

run-frontend: frontend-install
	cd demo/frontend && npm run dev

# ------------------------------------------------------------------------------
# Help
# ------------------------------------------------------------------------------
help:
	@echo "GPU Economic Telemetry Agent — Makefile targets"
	@echo ""
	@echo "  format        Format Go code (gofmt + goimports if installed)"
	@echo "  format-check  Check that Go code is formatted (fails if not; for CI)"
	@echo "  build         Build agent, script, and demo binaries"
	@echo "  build-agent   Build gpu-econ-agent only"
	@echo "  build-script  Build extract_gpu_snapshot only"
	@echo "  build-demo    Build demo ingestion service only"
	@echo "  test          Run tests"
	@echo "  lint          Run go vet"
	@echo "  tidy          Run go mod tidy"
	@echo "  clean         Remove built binaries and test cache"
	@echo "  run-agent     Build and run agent (CONFIG=path/to/config.yaml)"
	@echo "  run-demo      Build and run demo ingestion on :4317"
	@echo "  run-snapshot   Build and run extract_gpu_snapshot"
	@echo "  run-frontend   Install deps and run React analysis UI (demo/frontend)"
	@echo "  frontend-install  npm install in demo/frontend"
	@echo "  frontend-build    npm run build in demo/frontend"
	@echo "  all           format, build, test (default)"
	@echo "  help          Show this help"
	@echo ""
	@echo "Variables: CONFIG (for run-agent), VERSION (for build-agent)"

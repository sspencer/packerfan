GO              ?= go
BINARY          ?= pacfan
HEADLESS_BINARY ?= pacfan-headless
ARGS            ?=
GOFLAGS         ?=
COVER_FILE      ?= coverage.out
BENCH           ?= .
COUNT           ?= 1

.PHONY: build build-headless run test test-race bench cover fmt vet fix check check-headless-deps generate deps clean help

build: ## Build the binary
	$(GO) build $(GOFLAGS) -o $(BINARY) .

build-headless: ## Build the headless NDJSON simulation server
	$(GO) build $(GOFLAGS) -o $(HEADLESS_BINARY) ./cmd/pacfan-headless

run: build ## Build and run the binary (e.g. make run ARGS="-pacman -zoom 4")
	./$(BINARY) $(ARGS)

test: ## Run all tests
	$(GO) test $(GOFLAGS) ./...

test-race: ## Run all tests with the race detector enabled
	$(GO) test -race $(GOFLAGS) ./...

bench: ## Run simulation and API benchmarks (e.g. make bench BENCH=EnvStep COUNT=6)
	$(GO) test -run '^$$' -bench '$(BENCH)' -benchmem -count $(COUNT) ./internal/sim ./internal/api ./cmd/pacfan-headless

cover: ## Run tests and print total statement coverage
	$(GO) test -coverprofile=$(COVER_FILE) ./...
	$(GO) tool cover -func=$(COVER_FILE) | tail -n 1

fmt: ## Format Go source files
	$(GO) fmt ./...

vet: ## Run go vet static analysis
	$(GO) vet ./...

fix: ## Run go fix modernizers
	$(GO) fix ./...

check-headless-deps: ## Verify headless packages do not import Raylib, audio, scene, or render
	@if $(GO) list -deps ./internal/sim ./internal/api ./cmd/pacfan-headless | grep -E 'raylib|internal/audio|internal/scene|internal/render'; then \
		echo "ERROR: forbidden dependency found in headless packages"; \
		exit 1; \
	fi

check: fmt vet check-headless-deps test-race ## Run fmt, vet, dependency checks, and race-checked tests

generate: ## Run go generate (regenerate stringer enum helpers)
	$(GO) generate ./...

deps: ## Download and tidy module dependencies
	$(GO) mod download
	$(GO) mod tidy

clean: ## Remove build and coverage artifacts
	$(GO) clean
	rm -f $(BINARY) $(HEADLESS_BINARY) $(COVER_FILE)

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "; printf "Available targets:\n"} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

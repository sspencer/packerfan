GO              ?= go
BINARY          ?= pacfan
HEADLESS_BINARY ?= pacfan-headless
ARGS            ?=
GOFLAGS         ?=
COVER_FILE      ?= coverage.out
BENCH           ?= .
COUNT           ?= 1
BIN_DIR         ?= bin
PLATFORMS       ?= darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64

.PHONY: build build-headless build-all $(PLATFORMS) run test test-race bench cover fmt vet fix check check-headless-deps generate deps clean help

build: ## Build the binary
	$(GO) build $(GOFLAGS) -o $(BINARY) .

build-headless: ## Build the headless NDJSON simulation server
	$(GO) build $(GOFLAGS) -o $(HEADLESS_BINARY) ./cmd/pacfan-headless

build-all: $(PLATFORMS) ## Cross-compile game and headless binaries for mac, linux, and windows into $(BIN_DIR)/

os   = $(word 1,$(subst /, ,$@))
arch = $(word 2,$(subst /, ,$@))
ext  = $(if $(filter windows,$(os)),.exe,)

# Cross-compiles with CGO disabled: Ebitengine uses pure-Go bindings (purego,
# Win32/DirectX, pure-Go X11/OpenGL/PulseAudio), so no C cross-toolchain is needed.
$(PLATFORMS):
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=$(os) GOARCH=$(arch) $(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(BINARY)-$(os)-$(arch)$(ext) .
	CGO_ENABLED=0 GOOS=$(os) GOARCH=$(arch) $(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(HEADLESS_BINARY)-$(os)-$(arch)$(ext) ./cmd/pacfan-headless

run: build ## Build and run the binary (e.g. make run ARGS="--ms --zoom 4")
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

check-headless-deps: ## Verify headless packages do not import Ebitengine, Raylib, audio, scene, or render
	@if $(GO) list -deps ./internal/sim ./internal/api ./cmd/pacfan-headless | grep -E 'ebiten|raylib|internal/audio|internal/scene|internal/render'; then \
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
	rm -rf $(BIN_DIR)

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "; printf "Available targets:\n"} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

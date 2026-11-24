GO              ?= go
BINARY          ?= cheddarhead
HEADLESS_BINARY ?= cheddarhead-headless
ARGS            ?=
GOFLAGS         ?=
COVER_FILE      ?= coverage.out
BENCH           ?= .
COUNT           ?= 1
BIN_DIR         ?= bin
PLATFORMS       ?= darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64
APP_NAME        ?= CheddarHead
APP_BUNDLE_NAME ?= $(APP_NAME).app
APP_VERSION     ?= 1.0
BUNDLE_ID       ?= com.sspencer.cheddarhead
APP_BUNDLE      := $(BIN_DIR)/$(APP_BUNDLE_NAME)
APP_BUILD        = $(shell git rev-list --count HEAD 2>/dev/null || echo 1)
INSTALL_DIR     ?= /Applications
UNAME_S         := $(shell uname -s)
GOBIN           := $(shell $(GO) env GOBIN)
GOBIN_DIR       ?= $(if $(GOBIN),$(GOBIN),$(shell $(GO) env GOPATH)/bin)

.DEFAULT_GOAL := help

.PHONY: all build build-headless build-all $(PLATFORMS) app install install-app uninstall icon \
        run dev test test-race bench cover fmt fmt-check vet lint fix check-headless-deps check generate deps clean help

build: ## Build the game binary
	$(GO) build $(GOFLAGS) -o $(BINARY) .

build-headless: ## Build the headless NDJSON simulation server
	$(GO) build $(GOFLAGS) -o $(HEADLESS_BINARY) ./cmd/$(HEADLESS_BINARY)

build-all: $(PLATFORMS) ## Cross-compile game and headless binaries for mac, linux, and windows into $(BIN_DIR)/

all: build-all $(if $(filter Darwin,$(UNAME_S)),app) ## Cross-compile all binaries, plus the macOS app when run on macOS

os   = $(word 1,$(subst /, ,$@))
arch = $(word 2,$(subst /, ,$@))
ext  = $(if $(filter windows,$(os)),.exe,)

# Cross-compiles with CGO disabled: Ebitengine uses pure-Go bindings (purego,
# Win32/DirectX, pure-Go X11/OpenGL/PulseAudio), so no C cross-toolchain is needed.
$(PLATFORMS):
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=$(os) GOARCH=$(arch) $(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(BINARY)-$(os)-$(arch)$(ext) .
	CGO_ENABLED=0 GOOS=$(os) GOARCH=$(arch) $(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(HEADLESS_BINARY)-$(os)-$(arch)$(ext) ./cmd/$(HEADLESS_BINARY)

icon: ## Regenerate packaging/macos/icon.png (source for the app icon)
	$(GO) run ./tools/genicon packaging/macos/icon.png

app: ## Build a universal (arm64 + amd64) macOS app bundle at $(APP_BUNDLE)
ifneq ($(UNAME_S),Darwin)
	$(error make app builds a macOS .app bundle and must run on macOS)
endif
	rm -rf $(APP_BUNDLE) $(BIN_DIR)/AppIcon.iconset
	mkdir -p $(APP_BUNDLE)/Contents/MacOS $(APP_BUNDLE)/Contents/Resources $(BIN_DIR)/AppIcon.iconset
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BIN_DIR)/.app-arm64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BIN_DIR)/.app-amd64 .
	lipo -create -output $(APP_BUNDLE)/Contents/MacOS/$(BINARY) $(BIN_DIR)/.app-arm64 $(BIN_DIR)/.app-amd64
	rm -f $(BIN_DIR)/.app-arm64 $(BIN_DIR)/.app-amd64
	for s in 16 32 128 256 512; do \
		sips -z $$s $$s packaging/macos/icon.png --out $(BIN_DIR)/AppIcon.iconset/icon_$${s}x$${s}.png >/dev/null; \
		sips -z $$((s*2)) $$((s*2)) packaging/macos/icon.png --out $(BIN_DIR)/AppIcon.iconset/icon_$${s}x$${s}@2x.png >/dev/null; \
	done
	iconutil -c icns -o $(APP_BUNDLE)/Contents/Resources/AppIcon.icns $(BIN_DIR)/AppIcon.iconset
	rm -rf $(BIN_DIR)/AppIcon.iconset
	sed -e 's/@APP_NAME@/$(APP_NAME)/g' -e 's/@BUNDLE_ID@/$(BUNDLE_ID)/g' -e 's/@EXECUTABLE@/$(BINARY)/g' \
		-e 's/@VERSION@/$(APP_VERSION)/g' -e 's/@BUILD@/$(APP_BUILD)/g' \
		packaging/macos/Info.plist > $(APP_BUNDLE)/Contents/Info.plist
	plutil -lint $(APP_BUNDLE)/Contents/Info.plist >/dev/null
	codesign --force --sign - $(APP_BUNDLE)

ifeq ($(UNAME_S),Darwin)
install: app ## Install the game (macOS: $(INSTALL_DIR); elsewhere: GOBIN or GOPATH/bin)
	rm -rf "$(INSTALL_DIR)/$(APP_BUNDLE_NAME)"
	cp -R $(APP_BUNDLE) "$(INSTALL_DIR)/"

uninstall: ## Remove the installed game
	rm -rf "$(INSTALL_DIR)/$(APP_BUNDLE_NAME)"
else
install: build
	install -d "$(GOBIN_DIR)"
	install -m 0755 $(BINARY) "$(GOBIN_DIR)/$(BINARY)"

uninstall:
	rm -f "$(GOBIN_DIR)/$(BINARY)"
endif

install-app: install ## Alias for install (build and install the macOS app bundle)

run: build ## Build and run the binary (e.g. make run ARGS="--ms --zoom 4")
	./$(BINARY) $(ARGS)

dev: ## Run directly from source (e.g. make dev ARGS="--ms --zoom 4")
	$(GO) run . $(ARGS)

test: ## Run all tests
	$(GO) test $(GOFLAGS) ./...

test-race: ## Run all tests with the race detector enabled
	$(GO) test -race $(GOFLAGS) ./...

bench: ## Run simulation and API benchmarks (e.g. make bench BENCH=EnvStep COUNT=6)
	$(GO) test -run '^$$' -bench '$(BENCH)' -benchmem -count $(COUNT) ./internal/sim ./internal/api ./cmd/$(HEADLESS_BINARY)

cover: ## Run tests and print total statement coverage
	$(GO) test -coverprofile=$(COVER_FILE) ./...
	$(GO) tool cover -func=$(COVER_FILE) | tail -n 1

fmt: ## Format Go source files
	$(GO) fmt ./...

fmt-check: ## Fail if any Go source file is not formatted
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "Unformatted files:"; echo "$$out"; exit 1; fi

vet: ## Run go vet static analysis
	$(GO) vet ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

fix: ## Run go fix modernizers
	$(GO) fix ./...

check-headless-deps: ## Verify headless packages do not import Ebitengine, Raylib, audio, scene, or render
	@if $(GO) list -deps ./internal/sim ./internal/api ./cmd/$(HEADLESS_BINARY) | grep -E 'ebiten|raylib|internal/audio|internal/scene|internal/render'; then \
		echo "ERROR: forbidden dependency found in headless packages"; \
		exit 1; \
	fi

check: fmt-check vet check-headless-deps test-race ## Run fmt-check, vet, dependency checks, and race-checked tests

generate: ## Run go generate (regenerate stringer enum helpers)
	$(GO) generate ./...

deps: ## Download, tidy, and verify module dependencies
	$(GO) mod download
	$(GO) mod tidy
	$(GO) mod verify

clean: ## Remove build and coverage artifacts
	$(GO) clean
	rm -f $(BINARY) $(HEADLESS_BINARY) $(COVER_FILE) coverage.html *.test
	rm -rf $(BIN_DIR)

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "; printf "Available targets:\n"} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

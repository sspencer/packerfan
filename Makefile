GO         ?= go
BINARY     ?= pacfan
ARGS       ?=
GOFLAGS    ?=
COVER_FILE ?= coverage.out

.PHONY: build run test test-race cover fmt vet fix check generate deps clean help

build: ## Build the binary
	$(GO) build $(GOFLAGS) -o $(BINARY) .

run: build ## Build and run the binary (e.g. make run ARGS="-pacman -zoom 4")
	./$(BINARY) $(ARGS)

test: ## Run all tests
	$(GO) test $(GOFLAGS) ./...

test-race: ## Run all tests with the race detector enabled
	$(GO) test -race $(GOFLAGS) ./...

cover: ## Run tests and print total statement coverage
	$(GO) test -coverprofile=$(COVER_FILE) ./...
	$(GO) tool cover -func=$(COVER_FILE) | tail -n 1

fmt: ## Format Go source files
	$(GO) fmt ./...

vet: ## Run go vet static analysis
	$(GO) vet ./...

fix: ## Run go fix modernizers
	$(GO) fix ./...

check: fmt vet test-race ## Run fmt, vet, and race-checked tests

generate: ## Run go generate (regenerate stringer enum helpers)
	$(GO) generate ./...

deps: ## Download and tidy module dependencies
	$(GO) mod download
	$(GO) mod tidy

clean: ## Remove build and coverage artifacts
	$(GO) clean
	rm -f $(BINARY) $(COVER_FILE)

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "; printf "Available targets:\n"} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

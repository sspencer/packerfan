.PHONY: build run clean test fmt vet lint help

# Binary name
BINARY_NAME=pacfan

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build the project
build:
	$(GOBUILD) -o $(BINARY_NAME) -v

# Run the project
run: build
	./$(BINARY_NAME)

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

# Run tests
test:
	$(GOTEST) -v ./...

# Format code
fmt:
	$(GOFMT) ./...

# Run go vet
vet:
	$(GOVET) ./...

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run all checks (fmt, vet, test)
check: fmt vet test

# Show help
help:
	@echo "Available targets:"
	@echo "  build   - Build the binary"
	@echo "  run     - Build and run the binary"
	@echo "  clean   - Remove build artifacts"
	@echo "  test    - Run tests"
	@echo "  fmt     - Format code"
	@echo "  vet     - Run go vet"
	@echo "  deps    - Download and tidy dependencies"
	@echo "  check   - Run fmt, vet, and test"
	@echo "  help    - Show this help message"

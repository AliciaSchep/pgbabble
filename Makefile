.PHONY: build test clean install help

BINARY_NAME=pgbabble
BUILD_DIR=./build
MAIN_PATH=./cmd/pgbabble/main.go

# Default target
all: build

# Build the binary
build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

# Build with optimizations for release
build-release:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BINARY_NAME) $(MAIN_PATH)

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	go clean

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code and tidy modules
fmt:
	go fmt ./...
	go mod tidy

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Install the binary to GOPATH/bin
install:
	go install $(MAIN_PATH)

# Show help
help:
	@echo "Available targets:"
	@echo "  build           - Build the pgbabble binary"
	@echo "  build-release   - Build optimized binary for release"
	@echo "  test            - Run tests"
	@echo "  test-coverage   - Run tests with coverage"
	@echo "  test-verbose    - Run tests with verbose output"
	@echo "  clean           - Clean build artifacts"
	@echo "  deps            - Download and tidy dependencies"
	@echo "  fmt             - Format Go code"
	@echo "  lint            - Run linter (requires golangci-lint)"
	@echo "  install         - Install binary to GOPATH/bin"
	@echo "  help            - Show this help message"
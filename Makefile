.PHONY: build test clean install help test-db-start test-db-stop test-db-reset test-db-seed test-with-db

BINARY_NAME=pgbabble
BUILD_DIR=./build
MAIN_PATH=./cmd/pgbabble/main.go

# Test database configuration
TEST_DB_CONTAINER=pgbabble-test-db
TEST_DB_PORT=5433
TEST_DB_USER=testuser
TEST_DB_PASSWORD=testpass
TEST_DB_NAME=testdb

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
	@PGBABBLE_TEST_HOST=$${PGBABBLE_TEST_HOST:-localhost} \
	 PGBABBLE_TEST_PORT=$${PGBABBLE_TEST_PORT:-$(TEST_DB_PORT)} \
	 PGBABBLE_TEST_USER=$${PGBABBLE_TEST_USER:-$(TEST_DB_USER)} \
	 PGBABBLE_TEST_PASSWORD=$${PGBABBLE_TEST_PASSWORD:-$(TEST_DB_PASSWORD)} \
	 PGBABBLE_TEST_DATABASE=$${PGBABBLE_TEST_DATABASE:-$(TEST_DB_NAME)} \
	 go test ./...

# Run tests with coverage
test-coverage:
	@PGBABBLE_TEST_HOST=$${PGBABBLE_TEST_HOST:-localhost} \
	 PGBABBLE_TEST_PORT=$${PGBABBLE_TEST_PORT:-$(TEST_DB_PORT)} \
	 PGBABBLE_TEST_USER=$${PGBABBLE_TEST_USER:-$(TEST_DB_USER)} \
	 PGBABBLE_TEST_PASSWORD=$${PGBABBLE_TEST_PASSWORD:-$(TEST_DB_PASSWORD)} \
	 PGBABBLE_TEST_DATABASE=$${PGBABBLE_TEST_DATABASE:-$(TEST_DB_NAME)} \
	 go test -cover ./...

# Run tests with verbose output
test-verbose:
	@PGBABBLE_TEST_HOST=$${PGBABBLE_TEST_HOST:-localhost} \
	 PGBABBLE_TEST_PORT=$${PGBABBLE_TEST_PORT:-$(TEST_DB_PORT)} \
	 PGBABBLE_TEST_USER=$${PGBABBLE_TEST_USER:-$(TEST_DB_USER)} \
	 PGBABBLE_TEST_PASSWORD=$${PGBABBLE_TEST_PASSWORD:-$(TEST_DB_PASSWORD)} \
	 PGBABBLE_TEST_DATABASE=$${PGBABBLE_TEST_DATABASE:-$(TEST_DB_NAME)} \
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

# Start test database
test-db-start:
	@echo "Starting PostgreSQL test database..."
	@if docker ps -a --format 'table {{.Names}}' | grep -q "^$(TEST_DB_CONTAINER)$$"; then \
		echo "Container $(TEST_DB_CONTAINER) already exists. Removing..."; \
		docker rm -f $(TEST_DB_CONTAINER); \
	fi
	@docker run --name $(TEST_DB_CONTAINER) \
		-e POSTGRES_PASSWORD=$(TEST_DB_PASSWORD) \
		-e POSTGRES_USER=$(TEST_DB_USER) \
		-e POSTGRES_DB=$(TEST_DB_NAME) \
		-p $(TEST_DB_PORT):5432 \
		-d postgres:17-alpine
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Test database started on port $(TEST_DB_PORT)"
	@echo "Connection: postgresql://$(TEST_DB_USER):$(TEST_DB_PASSWORD)@localhost:$(TEST_DB_PORT)/$(TEST_DB_NAME)"

# Stop test database
test-db-stop:
	@echo "Stopping test database..."
	@docker stop $(TEST_DB_CONTAINER) || true
	@docker rm $(TEST_DB_CONTAINER) || true
	@echo "Test database stopped and removed"

# Reset test database (stop and start)
test-db-reset: test-db-stop test-db-start

# Seed test database with schema and test data
test-db-seed:
	@echo "Seeding test database..."
	@PGBABBLE_TEST_HOST=$${PGBABBLE_TEST_HOST:-localhost} \
	 PGBABBLE_TEST_PORT=$${PGBABBLE_TEST_PORT:-$(TEST_DB_PORT)} \
	 PGBABBLE_TEST_USER=$${PGBABBLE_TEST_USER:-$(TEST_DB_USER)} \
	 PGBABBLE_TEST_PASSWORD=$${PGBABBLE_TEST_PASSWORD:-$(TEST_DB_PASSWORD)} \
	 PGBABBLE_TEST_DATABASE=$${PGBABBLE_TEST_DATABASE:-$(TEST_DB_NAME)} \
	 go run ./scripts/seed-test-db.go

# Show help
help:
	@echo "Available targets:"
	@echo "  build                 - Build the pgbabble binary"
	@echo "  build-release         - Build optimized binary for release"
	@echo "  test                  - Run tests (unit tests only)"
	@echo "  test-coverage         - Run tests with coverage (unit tests only)"
	@echo "  test-verbose          - Run tests with verbose output (unit tests only)"
	@echo "  test-db-start         - Start PostgreSQL test database container"
	@echo "  test-db-stop          - Stop and remove test database container"
	@echo "  test-db-reset         - Stop and restart test database"
	@echo "  test-db-seed          - Seed test database with schema and test data"
	@echo "  test-with-db          - Run all tests with real database"
	@echo "  test-with-db-coverage - Run all tests with real database and coverage"
	@echo "  clean                 - Clean build artifacts"
	@echo "  deps                  - Download and tidy dependencies"
	@echo "  fmt                   - Format Go code"
	@echo "  lint                  - Run linter (requires golangci-lint)"
	@echo "  install               - Install binary to GOPATH/bin"
	@echo "  help                  - Show this help message"
# Variables
BINARY_NAME=server
BINARY_PATH=bin/$(BINARY_NAME)
MAIN_PATH=cmd/server/main.go
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Build variables
BUILD_FLAGS=-v
LDFLAGS=-ldflags="-s -w"

.PHONY: setup install-tools fmt fmt-check vet lint tidy tidy-check unit-test integration-test e2e-test test-all test-coverage deps build run dev clean wire help

setup: install-tools
install-tools:
	@echo "Installing development tools..."
	go mod download
	go install github.com/air-verse/air@v1.62.0
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0
	go install github.com/google/wire/cmd/wire@v0.7.0

fmt:
	@echo "Formatting code..."
	gofmt -w .
	@echo "Done"

fmt-check:
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "The following files need formatting:"; \
		gofmt -l .; \
		exit 1; \
	fi
	@echo "Done"

vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "Done"

lint:
	@echo "Running golangci-lint..."
	golangci-lint run
	@echo "Done"

tidy:
	@echo "Tidying go modules..."
	go mod tidy
	@echo "Done"

tidy-check:
	@echo "Checking if go.mod is tidy..."
	go mod tidy
	@if ! git diff --quiet go.mod go.sum; then \
		echo "go.mod or go.sum is not tidy"; \
		git diff go.mod go.sum; \
		exit 1; \
	fi
	@echo "Done"

unit-test:
	@echo "Running tests..."
	go test -v -race ./... -short
	@echo "Done"

# Additional commands from LP-247
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod verify
	@echo "Done"

build:
	@echo "Building $(BINARY_NAME)..."
	go build $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "Done"

run:
	@echo "Running server..."
	go run $(MAIN_PATH)

dev:
	@echo "Running server with hot reload..."
	air

clean:
	@echo "Cleaning up..."
	go clean
	rm -rf bin/ tmp/ $(COVERAGE_FILE) $(COVERAGE_HTML)
	@echo "Done"

integration-test:
	@echo "Running integration tests..."
	go test ./test/integration/... -v -race -timeout 60s
	@echo "Done"

e2e-test:
	@echo "Running E2E tests..."
	go test ./test/e2e/... -v -race -timeout 120s
	@echo "Done"

test-all: unit-test integration-test e2e-test

test-coverage:
	@echo "Running tests with coverage..."
	go test ./... -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic
	go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"
	@go tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print "Total Coverage: " $$3}'

wire:
	@echo "Generating wire dependencies..."
	wire ./cmd/server
	@echo "Done"

help:
	@echo "Available targets:"
	@echo "  setup         - Install development tools and dependencies"
	@echo "  install-tools - Install required development tools"
	@echo "  fmt          - Format code"
	@echo "  fmt-check    - Check code formatting"
	@echo "  vet          - Run go vet"
	@echo "  lint         - Run linter"
	@echo "  tidy         - Tidy go modules"
	@echo "  tidy-check   - Check if go.mod is tidy"
	@echo "  unit-test    - Run unit tests"
	@echo "  integration-test - Run integration tests"
	@echo "  e2e-test     - Run E2E tests"
	@echo "  test-all     - Run all tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  deps         - Download and verify dependencies"
	@echo "  build        - Build the binary"
	@echo "  run          - Run the application"
	@echo "  dev          - Run with hot reload"
	@echo "  clean        - Clean build artifacts"
	@echo "  wire         - Generate wire dependencies"
	@echo "  help         - Show this help message"

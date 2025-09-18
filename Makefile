.PHONY: setup install-tools fmt fmt-check vet lint tidy tidy-check test test-all

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

test:
	@echo "Running tests..."
	go test -v -race ./...
	@echo "Done"

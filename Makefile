.PHONY: setup install-tools

setup: install-tools

install-tools:
	@echo "Installing development tools..."
	go mod download
	go install github.com/air-verse/air@v1.62.0
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0
	go install github.com/google/wire/cmd/wire@v0.7.0

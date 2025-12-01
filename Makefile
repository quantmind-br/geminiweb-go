# Makefile for geminiweb Go CLI

BINARY_NAME=geminiweb
BUILD_DIR=build
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X github.com/diogo/geminiweb/internal/commands.Version=$(VERSION) -X github.com/diogo/geminiweb/internal/commands.BuildTime=$(BUILD_TIME)"

.PHONY: all build clean test lint fmt deps install run

all: build

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build the binary
build: deps
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/geminiweb

# Build for development (faster, no optimization)
build-dev: deps
	CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/geminiweb

# Install to GOPATH/bin
install: deps
	CGO_ENABLED=1 go install $(LDFLAGS) ./cmd/geminiweb

# Run the CLI
run: build-dev
	./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Remove temporary/useless files
clean-repo: clean
	rm -f plan-tests.md coverage-plan.md test-coverage-improvement-report.md

# Run linter
lint:
	golangci-lint run ./...

# Show coverage breakdown by function
cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

# Format code
fmt:
	go fmt ./...
	gofumpt -w .

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Build for all platforms (requires goreleaser)
release-snapshot:
	goreleaser release --snapshot --clean

# Check if build would succeed
check:
	go build -o /dev/null ./cmd/geminiweb

# Verify go.mod is tidy
verify-mod:
	go mod tidy
	git diff --exit-code go.mod go.sum

# Help
help:
	@echo "Available targets:"
	@echo "  deps            Download dependencies"
	@echo "  build           Build the binary"
	@echo "  build-dev       Build for development (faster)"
	@echo "  install         Install to GOPATH/bin"
	@echo "  run ARGS=...    Build and run with arguments"
	@echo "  test            Run tests"
	@echo "  test-coverage   Run tests with coverage report"
	@echo "  lint            Run linter"
	@echo "  clean-repo      Remove temporary/useless files           "
	@echo "(coverage reports, plans, build dir)"
	@echo "  fmt             Format code"
	@echo "  clean           Remove build artifacts"
	@echo "  release-snapshot Build for all platforms"
	@echo "  check           Verify build would succeed"
	@echo "  verify-mod      Verify go.mod is tidy"

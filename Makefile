# Project variables
BINARY_NAME=generator
BUILD_TIME=$(shell date +%s)
GIT_COMMIT=$(shell git rev-parse --short HEAD)

# Go variables
GO=go
GOFMT=gofmt
GOLINT=golangci-lint
GO_TEST_FLAGS=-v -race -cover

# Directories
SRC_DIR=.
CMD_DIR=./cmd
BUILD_DIR=./build
COVERAGE_DIR=./coverage

# Build flags
LDFLAGS=-ldflags "\
	-X main.Version=$(VERSION) \
	-X main.Commit=$(GIT_COMMIT) \
	-X main.BuildTime=$(BUILD_TIME) \
	-w -s"

VERSION=$(shell $(GO) run  $(CMD_DIR) -v | awk '{print $$NF}' || echo "v0.0.0")

# Default target
.PHONY: all
all: build

# Build the application
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_$(VERSION) $(CMD_DIR)

# Install dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Run the application
.PHONY: run
run:
	$(GO) run $(CMD_DIR)

# Test the application
.PHONY: test
test:
	@echo "Running tests..."
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test $(GO_TEST_FLAGS) -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html

# Run tests with coverage
.PHONY: test-coverage
test-coverage: test
	@echo "Coverage report generated at $(COVERAGE_DIR)/coverage.html"
	$(GO) tool cover -func=$(COVERAGE_DIR)/coverage.out

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
	rm -f *.log
	$(GO) clean

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) -w $(SRC_DIR)

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	$(GOLINT) run

# Run vet
.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

# Security check
.PHONY: security
security:
	@echo "Running security check..."
	$(GO) list -json -m all | nancy sleuth

# Build for Linux
.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_$(VERSION)-linux-amd64 $(CMD_DIR)

# Build for Windows
.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)_$(VERSION)-windows-amd64.exe $(CMD_DIR)

# Build all platforms
.PHONY: build-all
build-all: build-linux build-windows

# Install golangci-lint (if not installed)
.PHONY: install-lint
install-lint:
	@echo "Installing golangci-lint..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Generate CDR (example target - adjust according to your project)
.PHONY: generate-cdr
generate-cdr: build
	@echo "Generating CDR..."
	$(BUILD_DIR)/$(BINARY_NAME) -file -debug -rm -thread

.PHONY: release
release: clean build-linux build-windows
	@cd $(BUILD_DIR) && \
		tar -czf $(BINARY_NAME)-linux-$(VERSION).tar.gz $(BINARY_NAME)_$(VERSION)-linux-amd64 && \
		zip $(BINARY_NAME)-windows-$(VERSION).zip $(BINARY_NAME)_$(VERSION)-windows-amd64.exe

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo ""
	@echo "  all            - Build the application (default)"
	@echo "  build          - Build the application"
	@echo "  deps           - Download and tidy dependencies"
	@echo "  run            - Run the application"
	@echo "  test           - Run tests with coverage"
	@echo "  test-coverage  - Run tests and show coverage"
	@echo "  clean          - Clean build artifacts"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code (requires golangci-lint)"
	@echo "  vet            - Run go vet"
	@echo "  security       - Run security check (requires nancy)"
	@echo "  build-linux    - Build for Linux"
	@echo "  build-windows  - Build for Windows"
	@echo "  build-all      - Build for all platforms"
	@echo "  install-lint   - Install golangci-lint"
	@echo "  generate-cdr   - Generate CDR (example)"
	@echo "  help           - Show this help"
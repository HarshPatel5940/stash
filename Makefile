.PHONY: build install clean test run help

# Variables
BINARY_NAME=stash
INSTALL_PATH=/usr/local/bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "✓ Build complete: ./$(BINARY_NAME)"

install: build ## Install to /usr/local/bin
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@sudo mv $(BINARY_NAME) $(INSTALL_PATH)/
	@echo "✓ Installed to $(INSTALL_PATH)/$(BINARY_NAME)"

uninstall: ## Remove from /usr/local/bin
	@echo "Removing $(INSTALL_PATH)/$(BINARY_NAME)..."
	@sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✓ Uninstalled"

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@go clean
	@echo "✓ Clean complete"

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

run: build ## Build and run
	@./$(BINARY_NAME)

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Format complete"

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ Vet complete"

lint: fmt vet ## Run formatting and vetting

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies ready"

release: clean test build ## Clean, test, and build for release
	@echo "✓ Release build complete"

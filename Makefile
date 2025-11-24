# Enterprise GenAI Toolbox - Build System
# Cross-platform binary builds for easy enterprise distribution

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY_NAME = genai-toolbox
BUILD_DIR = dist
LDFLAGS = -ldflags "-X main.version=$(VERSION) -s -w"

# Build targets
.PHONY: all clean build build-all install test help

all: build

help: ## Show this help message
	@echo "Enterprise GenAI Toolbox - Build Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@echo "Clean complete!"

test: ## Run all tests
	@echo "Running tests..."
	@go test ./internal/sources/... -v

build: ## Build for current platform
	@echo "Building $(BINARY_NAME) for current platform..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Build complete: ./$(BINARY_NAME)"

install: build ## Install to $GOPATH/bin
	@echo "Installing $(BINARY_NAME) to $$GOPATH/bin..."
	@go install $(LDFLAGS) .
	@echo "Install complete!"

# Cross-platform builds
build-all: clean build-linux build-darwin build-windows ## Build for all platforms
	@echo ""
	@echo "All binaries built successfully!"
	@echo "Binaries available in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/

build-linux: ## Build for Linux (amd64 and arm64)
	@echo "Building for Linux amd64..."
	@mkdir -p $(BUILD_DIR)/linux/amd64
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/linux/amd64/$(BINARY_NAME) .
	@echo "Building for Linux arm64..."
	@mkdir -p $(BUILD_DIR)/linux/arm64
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/linux/arm64/$(BINARY_NAME) .

build-darwin: ## Build for macOS (amd64 and arm64)
	@echo "Building for macOS amd64 (Intel)..."
	@mkdir -p $(BUILD_DIR)/darwin/amd64
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/darwin/amd64/$(BINARY_NAME) .
	@echo "Building for macOS arm64 (Apple Silicon)..."
	@mkdir -p $(BUILD_DIR)/darwin/arm64
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/darwin/arm64/$(BINARY_NAME) .

build-windows: ## Build for Windows (amd64)
	@echo "Building for Windows amd64..."
	@mkdir -p $(BUILD_DIR)/windows/amd64
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/windows/amd64/$(BINARY_NAME).exe .

# Create release packages
package: build-all ## Create release packages (zip/tar.gz)
	@echo "Creating release packages..."
	@cd $(BUILD_DIR)/linux/amd64 && tar -czf ../../$(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)
	@cd $(BUILD_DIR)/linux/arm64 && tar -czf ../../$(BINARY_NAME)-linux-arm64.tar.gz $(BINARY_NAME)
	@cd $(BUILD_DIR)/darwin/amd64 && tar -czf ../../$(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)
	@cd $(BUILD_DIR)/darwin/arm64 && tar -czf ../../$(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)
	@cd $(BUILD_DIR)/windows/amd64 && zip -q ../../$(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME).exe
	@echo ""
	@echo "Release packages created:"
	@ls -lh $(BUILD_DIR)/*.{tar.gz,zip} 2>/dev/null || true

# Development
dev: ## Build and run locally
	@go run . --tools-file examples/tools.yaml

format: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete!"

lint: ## Run linters
	@echo "Running linters..."
	@golangci-lint run
	@echo "Lint complete!"

# Docker
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):$(VERSION) .
	@docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest
	@echo "Docker image built: $(BINARY_NAME):$(VERSION)"

docker-run: docker-build ## Run in Docker
	@docker run -p 5000:5000 -v $(PWD)/examples/tools.yaml:/app/tools.yaml $(BINARY_NAME):latest --tools-file /app/tools.yaml

# Enterprise distribution
release-info: ## Show release information
	@echo "Version: $(VERSION)"
	@echo "Binary Name: $(BINARY_NAME)"
	@echo "Build Directory: $(BUILD_DIR)"
	@echo ""
	@echo "Supported platforms:"
	@echo "  - Linux (amd64, arm64)"
	@echo "  - macOS (amd64/Intel, arm64/Apple Silicon)"
	@echo "  - Windows (amd64)"

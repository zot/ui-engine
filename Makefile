# UI Platform Makefile
# Spec: deployment.md, demo.md

# Build configuration
BINARY_NAME := ui
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Go configuration
GO := go
GOFLAGS := -trimpath

# Output directories
BUILD_DIR := build
RELEASE_DIR := release

# Frontend build
WEB_DIR := web
SITE_DIR := site

.PHONY: all build clean test lint fmt vet deps frontend release release-bundled demo bundle help

# Default target - build unbundled binary
all: deps frontend build

# Build unbundled binary for current platform
build:
	@echo "Building $(BINARY_NAME) (unbundled)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ui

# Build with SQLite support (requires CGO)
build-sqlite:
	@echo "Building $(BINARY_NAME) with SQLite (CGO enabled)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ui

# Build bundled binary for current platform
bundle: build frontend
	@echo "Creating bundled binary..."
	@mkdir -p $(BUILD_DIR)/bundle_root
	@cp -r $(SITE_DIR)/* $(BUILD_DIR)/bundle_root/
	@mkdir -p $(BUILD_DIR)/bundle_root/resources
	@if [ -d "resources" ]; then cp -r resources/* $(BUILD_DIR)/bundle_root/resources/; fi
	$(BUILD_DIR)/$(BINARY_NAME) bundle -o $(BUILD_DIR)/$(BINARY_NAME)-bundled $(BUILD_DIR)/bundle_root
	@rm -rf $(BUILD_DIR)/bundle_root
	@echo "Created: $(BUILD_DIR)/$(BINARY_NAME)-bundled"

# Build MCP-optimized binary (alias for bundle)
mcp: bundle

# Build frontend
frontend:
	@echo "Building frontend..."
	@cd $(WEB_DIR) && npm install && npm run build

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) $(RELEASE_DIR)
	@$(GO) clean -cache -testcache

# Run tests
test:
	@echo "Running tests..."
	CGO_ENABLED=0 $(GO) test -v ./...

# Run tests with race detector (requires CGO)
test-race:
	@echo "Running tests with race detector..."
	CGO_ENABLED=1 $(GO) test -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) test -v -coverprofile=$(BUILD_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report: $(BUILD_DIR)/coverage.html"

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run ./... || echo "Install golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Run go vet
vet:
	@echo "Running vet..."
	$(GO) vet ./...

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) work sync

# Build unbundled release binaries for all platforms
release: frontend
	@echo "Building release binaries (unbundled)..."
	@mkdir -p $(RELEASE_DIR)
	@# Linux AMD64
	@echo "  Building linux/amd64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(RELEASE_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/ui
	@# Linux ARM64
	@echo "  Building linux/arm64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(RELEASE_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/ui
	@# macOS AMD64
	@echo "  Building darwin/amd64..."
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(RELEASE_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/ui
	@# macOS ARM64 (Apple Silicon)
	@echo "  Building darwin/arm64..."
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(RELEASE_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/ui
	@# Windows AMD64
	@echo "  Building windows/amd64..."
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(RELEASE_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/ui
	@echo "Release binaries (unbundled) in $(RELEASE_DIR)/"
	@ls -la $(RELEASE_DIR)/

# Build bundled release binaries (bundles site into each platform binary)
release-bundled: build release
	@echo "Bundling site into release binaries..."
	@# Bundle each platform's binary using the local build with -src to specify target
	@for binary in $(RELEASE_DIR)/$(BINARY_NAME)-*; do \
		if [ -f "$$binary" ]; then \
			echo "  Bundling $$binary..."; \
			$(BUILD_DIR)/$(BINARY_NAME) bundle -src "$$binary" -o "$$binary.tmp" $(SITE_DIR) 2>/dev/null && \
			mv "$$binary.tmp" "$$binary" || \
			echo "  (skipping - failed to bundle)"; \
		fi; \
	done
	@echo "Bundled release binaries in $(RELEASE_DIR)/"
	@ls -la $(RELEASE_DIR)/

# Create release archives
release-archives: release-bundled
	@echo "Creating release archives..."
	@cd $(RELEASE_DIR) && \
		for f in $(BINARY_NAME)-*; do \
			if [ -f "$$f" ] && ! echo "$$f" | grep -q '\.\(tar\.gz\|zip\)$$'; then \
				if echo "$$f" | grep -q "windows"; then \
					zip -q "$${f%.exe}.zip" "$$f" && echo "  Created $${f%.exe}.zip"; \
				else \
					tar -czf "$$f.tar.gz" "$$f" && echo "  Created $$f.tar.gz"; \
				fi; \
			fi; \
		done
	@echo "Release archives created"

# Build demo binary with demo site bundled
demo: build frontend
	@echo "Copying frontend to demo..."
	@cp $(SITE_DIR)/html/main.js $(SITE_DIR)/html/worker.js demo/html/
	@echo "Building demo binary..."
	$(BUILD_DIR)/$(BINARY_NAME) bundle -o $(BUILD_DIR)/$(BINARY_NAME)-demo demo
	@echo "Created: $(BUILD_DIR)/$(BINARY_NAME)-demo"

# Run the server (development - uses --dir for live reload)
run: build
	@echo "Starting server..."
	$(BUILD_DIR)/$(BINARY_NAME) serve --dir $(SITE_DIR)

# Run the bundled server
run-bundled: bundle
	@echo "Starting bundled server..."
	$(BUILD_DIR)/$(BINARY_NAME)-bundled serve

# Run the demo
run-demo: demo
	@echo "Starting demo server..."
	$(BUILD_DIR)/$(BINARY_NAME)-demo serve --lua-path demo/lua

# Install to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	CGO_ENABLED=0 $(GO) install $(GOFLAGS) $(LDFLAGS) ./cmd/ui

# Check build requirements
check:
	@echo "Checking requirements..."
	@command -v go >/dev/null 2>&1 || { echo "Go is required but not installed."; exit 1; }
	@command -v npm >/dev/null 2>&1 || { echo "npm is required but not installed."; exit 1; }
	@echo "Go version: $$(go version)"
	@echo "npm version: $$(npm --version)"
	@echo "All requirements met"

# Show help
help:
	@echo "UI Platform Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build Targets:"
	@echo "  all             Build everything (deps, frontend, unbundled binary)"
	@echo "  build           Build unbundled binary for current platform"
	@echo "  build-sqlite    Build with SQLite support (CGO enabled)"
	@echo "  bundle          Build bundled binary (includes site)"
	@echo "  frontend        Build frontend TypeScript"
	@echo "  demo            Build demo binary with demo site bundled"
	@echo ""
	@echo "Release Targets:"
	@echo "  release         Build unbundled binaries for all platforms"
	@echo "  release-bundled Build bundled binaries for all platforms"
	@echo "  release-archives Create release archives (tar.gz, zip)"
	@echo ""
	@echo "Development Targets:"
	@echo "  run             Run server with --dir (live reload)"
	@echo "  run-bundled     Run bundled server"
	@echo "  run-demo        Run demo server"
	@echo "  test            Run tests"
	@echo "  test-race       Run tests with race detector (CGO)"
	@echo "  test-coverage   Run tests with coverage report"
	@echo ""
	@echo "Maintenance Targets:"
	@echo "  clean           Remove build artifacts"
	@echo "  deps            Install Go dependencies"
	@echo "  lint            Run linter"
	@echo "  fmt             Format Go code"
	@echo "  vet             Run go vet"
	@echo "  install         Install to GOPATH/bin"
	@echo "  check           Check build requirements"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION         Version string (default: git describe)"
	@echo ""
	@echo "Note: Bundled binaries contain the site embedded. Unbundled binaries"
	@echo "      require --dir to specify a site directory."

# Makefile for Shrew - Cross-compilation and Release Helper

# --- Variables ---
BINARY_NAME := shrew
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")
LDFLAGS := "-s -w -X main.version=$(VERSION)"

# --- Targets ---

.PHONY: all build clean install uninstall release-build

all: build

# Build for the current OS/ARCH
build:
	@echo "==> Building Shrew for $(shell go env GOOS)/$(shell go env GOARCH)..."
	@go build -ldflags=$(LDFLAGS) -o $(BINARY_NAME) .

# Clean up build artifacts
clean:
	@echo "==> Cleaning up..."
	@rm -f $(BINARY_NAME) shrew-*

# Install to /usr/local/bin
install: build
	@echo "==> Installing shrew to /usr/local/bin..."
	@sudo mv $(BINARY_NAME) /usr/local/bin/shrew

# Uninstall from /usr/local/bin
uninstall:
	@echo "==> Uninstalling shrew from /usr/local/bin..."
	@sudo rm -f /usr/local/bin/shrew

# Build binaries for release (Linux and Darwin)
release-build: clean
	@echo "==> Building release binaries..."
	@GOOS=linux GOARCH=amd64 go build -ldflags=$(LDFLAGS) -o shrew-linux-amd64 .
	@GOOS=linux GOARCH=arm64 go build -ldflags=$(LDFLAGS) -o shrew-linux-arm64 .
	@GOOS=darwin GOARCH=amd64 go build -ldflags=$(LDFLAGS) -o shrew-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 go build -ldflags=$(LDFLAGS) -o shrew-darwin-arm64 .
	@echo "==> Release binaries created:"
	@ls shrew-*

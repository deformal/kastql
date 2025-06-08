
BINARY_NAME := kastql

GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
DIST_DIR := dist/kastql_$(GOOS)_$(GOARCH)
BIN_PATH := $(DIST_DIR)/kastql

dev: build-dev install

build-dev:
	@echo "🔧 Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	goreleaser build --snapshot --clean
	@echo "Building the ui"
	cd ui && bun run build
		

install: build
	@echo "📦 Installing $(BINARY_NAME) to /usr/local/bin"
	cp $(BIN_PATH) /usr/local/bin/$(BINARY_NAME)
	chmod +x /usr/local/bin/$(BINARY_NAME)

uninstall:
	@echo "🧹 Uninstalling $(BINARY_NAME)"
	rm -f /usr/local/bin/$(BINARY_NAME)

version:
	@$(BINARY_NAME) --version

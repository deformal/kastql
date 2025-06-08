
BINARY_NAME := kastql

GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
DIST_DIR := dist/kastql_$(GOOS)_$(GOARCH)
BIN_PATH := $(DIST_DIR)/kastql

dev:
	./dist/kastql_darwin_arm64/kastql $(ARGS)

build-dev:
	@echo "🔧 Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	goreleaser build --snapshot --clean
	@echo "Building the ui"
	cd ui && bun run build
	sudo cp -r ./ui/out ./internal/ui/static/

release:
	@echo "Releasing"
	goreleaser release --clean
BINARY     := kastql
CMD        := ./cmd/kastql
IMAGE      := kastql
CONFIG     ?= config.yaml
WEB_DIR    := web
DIST_SRC   := $(WEB_DIR)/dist
DIST_EMBED := internal/playground/dist

# Admin panel uses the same web/ build — the SPA handles /admin/* routing
ADMIN_EMBED := internal/adminpanel/dist

# Version from git tag, falling back to "dev".
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS    := -s -w \
              -X main.version=$(VERSION) \
              -X main.commit=$(COMMIT) \
              -X main.buildTime=$(BUILD_TIME)

GO         := go
GOFLAGS    :=

.DEFAULT_GOAL := help

# ── Help ──────────────────────────────────────────────────────────────────────
# Prints every target that has a ## comment on the same line.
.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' \
		| sort

# ── Frontend ──────────────────────────────────────────────────────────────────
.PHONY: web
web: ## Build the GraphiQL playground (npm run build → embeds into binary)
	cd $(WEB_DIR) && npm install --silent && npm run build
	rm -rf $(DIST_EMBED)
	cp -r $(DIST_SRC) $(DIST_EMBED)

.PHONY: web-dev
web-dev: ## Start the Vite dev server (proxies /graphql and /v1/* to :8080)
	cd $(WEB_DIR) && npm run dev

.PHONY: web-install
web-install: ## Install frontend npm dependencies only
	cd $(WEB_DIR) && npm install

# ── Build ─────────────────────────────────────────────────────────────────────
.PHONY: build
build: web ## Build frontend then compile binary → ./kastql
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -trimpath -o $(BINARY) $(CMD)

.PHONY: build-no-web
build-no-web: ## Build binary only — skips frontend rebuild (uses existing dist)
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -trimpath -o $(BINARY) $(CMD)

.PHONY: build-linux
build-linux: web ## Cross-compile a static Linux/amd64 binary → ./kastql-linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -trimpath -o $(BINARY)-linux $(CMD)

.PHONY: build-linux-arm64
build-linux-arm64: web ## Cross-compile a static Linux/arm64 binary → ./kastql-linux-arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -trimpath -o $(BINARY)-linux-arm64 $(CMD)

# ── Run ───────────────────────────────────────────────────────────────────────
.PHONY: run
run: ## Run locally (skips frontend rebuild — use make web first if needed)
	$(GO) run $(CMD) -config $(CONFIG)

.PHONY: run-binary
run-binary: build ## Build (including frontend) then run the binary
	./$(BINARY) -config $(CONFIG)

# ── Test ──────────────────────────────────────────────────────────────────────
.PHONY: test
test: ## Run all tests
	$(GO) test ./...

.PHONY: test-v
test-v: ## Run all tests (verbose)
	$(GO) test -v ./...

.PHONY: test-race
test-race: ## Run tests with race detector
	$(GO) test -race ./...

.PHONY: cover
cover: ## Run tests and open HTML coverage report
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out

.PHONY: cover-text
cover-text: ## Print per-package coverage summary to terminal
	$(GO) test -coverprofile=coverage.out ./... && \
		$(GO) tool cover -func=coverage.out

# ── Code quality ──────────────────────────────────────────────────────────────
.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: fmt
fmt: ## Format all Go source files
	$(GO) fmt ./...

.PHONY: fmt-check
fmt-check: ## Check formatting without modifying files (CI-friendly)
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then \
		echo "Unformatted files:"; \
		echo "$$out"; \
		exit 1; \
	fi

.PHONY: lint
lint: ## Run golangci-lint (must be installed: brew install golangci-lint)
	golangci-lint run ./...

.PHONY: check
check: fmt-check vet test ## Run fmt-check + vet + tests (CI gate)

# ── Dependencies ──────────────────────────────────────────────────────────────
.PHONY: deps
deps: ## Tidy and verify go.mod / go.sum
	$(GO) mod tidy
	$(GO) mod verify

.PHONY: deps-upgrade
deps-upgrade: ## Upgrade all dependencies to latest minor/patch
	$(GO) get -u ./...
	$(GO) mod tidy

# ── Docker ────────────────────────────────────────────────────────────────────
.PHONY: docker-build
docker-build: ## Build the Docker image (tag: kastql:latest and kastql:VERSION)
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t $(IMAGE):latest \
		-t $(IMAGE):$(VERSION) \
		.

.PHONY: docker-run
docker-run: docker-build ## Build image and run a container (maps port 8080)
	docker run --rm -p 8080:8080 \
		-v "$(PWD)/config.yaml:/etc/kastql/config.yaml:ro" \
		-v kastql-data:/data \
		$(IMAGE):latest

.PHONY: up
up: ## Start services with docker compose
	docker compose up --build

.PHONY: up-detached
up-detached: ## Start services in the background
	docker compose up --build -d

.PHONY: down
down: ## Stop and remove compose containers
	docker compose down

.PHONY: logs
logs: ## Tail compose logs
	docker compose logs -f kastql

# ── Database ──────────────────────────────────────────────────────────────────
.PHONY: db-reset
db-reset: ## Delete local SQLite databases (forces fresh migration on next run)
	@echo "Deleting metadata.db and metrics.db …"
	rm -f metadata.db metadata.db-shm metadata.db-wal
	rm -f metrics.db  metrics.db-shm  metrics.db-wal

.PHONY: db-shell-meta
db-shell-meta: ## Open a SQLite shell on metadata.db (requires sqlite3)
	sqlite3 metadata.db

.PHONY: db-shell-metrics
db-shell-metrics: ## Open a SQLite shell on metrics.db (requires sqlite3)
	sqlite3 metrics.db

# ── Misc ──────────────────────────────────────────────────────────────────────
.PHONY: clean
clean: ## Remove build artifacts, coverage files, and embedded dist
	rm -f $(BINARY) $(BINARY)-linux $(BINARY)-linux-arm64
	rm -f coverage.out
	rm -rf $(DIST_EMBED) $(DIST_SRC)

.PHONY: version
version: ## Print the current version string
	@echo $(VERSION)

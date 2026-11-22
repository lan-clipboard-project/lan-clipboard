# LAN Clipboard — build & development helpers
#
#   make build   build the frontend and the Go binary
#   make run     build everything, then serve on http://localhost:8080
#   make test    run Go tests + frontend type-check
#   make smoke   build, then run an end-to-end smoke test
#   make help    list every target
#
# The Go toolchain defaults to whatever `go` is on PATH, falling back to the
# project-local toolchain in .tools/go (created by `make bootstrap-go`).
GO ?= $(shell command -v go 2>/dev/null || echo $(CURDIR)/.tools/go/bin/go)

GOTOOLDIR ?= $(CURDIR)/.tools
NPM ?= npm

BIN_DIR := bin
BIN := $(BIN_DIR)/lan-clipboard
WEB_DIR := web
SERVER_DIR := server
STATIC_DIR := $(SERVER_DIR)/static

# When using the project-local Go toolchain, keep its caches inside .tools/ so
# the build never needs write access to the user's home directory.
ifeq ($(GO),$(CURDIR)/.tools/go/bin/go)
GOCACHE ?= $(GOTOOLDIR)/gocache
GOPATH ?= $(GOTOOLDIR)/gopath
GO_ENV := GOCACHE="$(GOCACHE)" GOPATH="$(GOPATH)" GOENV=off GOTOOLCHAIN=auto
else
GO_ENV :=
endif

# go-sqlite3 only compiles its bundled SQLite with FTS5 when this build tag is set.
SQLITE_TAGS := sqlite_fts5

.PHONY: help bootstrap-go dev dev-server dev-web build build-web build-server \
	static run test test-server typecheck fmt vet smoke test-container clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

bootstrap-go: ## Download a project-local Go toolchain into .tools/ (no system changes)
	@if [ -x "$(GOTOOLDIR)/go/bin/go" ]; then \
		echo "Go toolchain already present: $$("$(GOTOOLDIR)/go/bin/go" version)"; \
	else \
		mkdir -p "$(GOTOOLDIR)"; \
		os=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		arch=$$(uname -m); \
		case "$$arch" in x86_64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; esac; \
		ver=$$(curl -fsSL 'https://go.dev/VERSION?m=text' | head -1); \
		echo "Downloading $$ver ($$os-$$arch) to $(GOTOOLDIR) ..."; \
		curl -fL --retry 2 -o "$(GOTOOLDIR)/go.tgz" "https://go.dev/dl/$$ver.$$os-$$arch.tar.gz"; \
		tar -xzf "$(GOTOOLDIR)/go.tgz" -C "$(GOTOOLDIR)"; \
		rm -f "$(GOTOOLDIR)/go.tgz"; \
		"$(GOTOOLDIR)/go/bin/go" version; \
	fi

build: build-web build-server ## Build the frontend and the Go binary

build-web: ## Build the web frontend into web/dist
	cd "$(WEB_DIR)" && $(NPM) run build

build-server: ## Compile the Go server into bin/lan-clipboard
	mkdir -p $(BIN_DIR)
	cd "$(SERVER_DIR)" && $(GO_ENV) "$(GO)" build -tags $(SQLITE_TAGS) -trimpath -o ../$(BIN) .

static: build-web ## Copy the built frontend into server/static (served by the Go server)
	rm -rf "$(STATIC_DIR)"
	cp -R "$(WEB_DIR)/dist" "$(STATIC_DIR)"

run: static build-server ## Build everything and run the server on http://localhost:8080
	cd $(SERVER_DIR) && ../$(BIN)

dev: ## Print the two commands needed for development
	@echo "Run these in two terminals:"
	@echo "  make dev-server   # API + WebSocket on :8080"
	@echo "  make dev-web      # Vite dev server, proxies /api and /ws to :8080"

dev-server: ## Run the API server directly from source
	cd "$(SERVER_DIR)" && $(GO_ENV) "$(GO)" run -tags $(SQLITE_TAGS) .

dev-web: ## Run the Vite dev server (proxies /api and /ws to :8080)
	cd "$(WEB_DIR)" && $(NPM) run dev

test: test-server typecheck ## Run Go tests and the frontend type-check

test-server: ## Run all Go tests
	cd "$(SERVER_DIR)" && $(GO_ENV) "$(GO)" test -tags $(SQLITE_TAGS) ./...

typecheck: ## Type-check the frontend
	cd "$(WEB_DIR)" && $(NPM) run typecheck

fmt: ## Format Go sources
	cd "$(SERVER_DIR)" && "$(GO)" fmt ./...

vet: ## Run go vet
	cd "$(SERVER_DIR)" && $(GO_ENV) "$(GO)" vet -tags $(SQLITE_TAGS) ./...

smoke: static build-server ## Build, then run the end-to-end smoke test
	bash scripts/smoke.sh

test-container: ## Test an existing image (IMAGE=lan-clipboard:ci); requires Docker Compose v2
	$(GO_ENV) GO="$(GO)" bash scripts/test-container.sh $(or $(IMAGE),lan-clipboard:ci)

clean: ## Remove build artifacts and local data
	rm -rf $(BIN_DIR) $(WEB_DIR)/dist $(STATIC_DIR) $(SERVER_DIR)/data

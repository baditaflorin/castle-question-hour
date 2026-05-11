.DEFAULT_GOAL := help
SHELL := /bin/bash

# ----- meta -----
APP            ?= castle-question-hour
GHCR_REPO      ?= ghcr.io/baditaflorin/$(APP)
VERSION        ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
PAGES_BASE     ?= /castle-question-hour/
BUILD_DIR      := docs
FRONTEND_DIR   := frontend
BACKEND_DIR    := backend
DEPLOY_DIR     := deploy

.PHONY: help
help: ## Show available targets
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# ----- bootstrap -----
.PHONY: install-hooks
install-hooks: ## Wire local git hooks (.githooks/)
	git config core.hooksPath .githooks
	chmod +x .githooks/* 2>/dev/null || true
	@echo "git hooks installed."

.PHONY: install
install: ## Install backend & frontend deps
	cd $(BACKEND_DIR) && go mod download
	cd $(FRONTEND_DIR) && npm ci

# ----- dev -----
.PHONY: dev
dev: ## Run backend (:8080) and frontend dev server (:5173) together
	@trap 'kill 0' SIGINT; \
	(cd $(BACKEND_DIR) && go run ./cmd/server) & \
	(cd $(FRONTEND_DIR) && npm run dev) & \
	wait

.PHONY: dev-backend
dev-backend: ## Run only the backend
	cd $(BACKEND_DIR) && go run ./cmd/server

.PHONY: dev-frontend
dev-frontend: ## Run only the frontend dev server
	cd $(FRONTEND_DIR) && npm run dev

# ----- build -----
.PHONY: build
build: build-backend build-frontend ## Build everything

.PHONY: build-frontend
build-frontend: ## Build frontend into ./docs (Pages-ready)
	cd $(FRONTEND_DIR) && npm ci && VITE_APP_VERSION=$(VERSION) BASE_PATH=$(PAGES_BASE) npm run build
	@test -f $(BUILD_DIR)/index.html || (echo "ERROR: build did not produce $(BUILD_DIR)/index.html" && exit 1)
	@cp $(BUILD_DIR)/index.html $(BUILD_DIR)/404.html
	@echo "frontend built into $(BUILD_DIR)/ (Pages-ready)"

.PHONY: build-backend
build-backend: ## Build backend binary into backend/bin/server
	cd $(BACKEND_DIR) && go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o bin/server ./cmd/server

# ----- test -----
.PHONY: test
test: ## Unit tests (backend + frontend)
	cd $(BACKEND_DIR) && go test -race -count=1 -coverprofile=coverage.out ./...
	cd $(FRONTEND_DIR) && npm test -- --run

.PHONY: test-integration
test-integration: ## Integration tests (gated by build tag)
	cd $(BACKEND_DIR) && go test -tags=integration -race -count=1 ./test/integration/...

.PHONY: smoke
smoke: ## Boot ephemeral compose, hit healthz/readyz/metrics, run Playwright happy-path
	./scripts/smoke.sh

# ----- quality -----
.PHONY: lint
lint: ## All linters
	cd $(BACKEND_DIR) && go vet ./... && (command -v golangci-lint >/dev/null && golangci-lint run || echo "golangci-lint not installed — skipping")
	cd $(FRONTEND_DIR) && npm run lint && npm run typecheck

.PHONY: fmt
fmt: ## Autoformat
	cd $(BACKEND_DIR) && gofmt -w . && (command -v goimports >/dev/null && goimports -w . || true)
	cd $(FRONTEND_DIR) && npm run fmt

# ----- pages preview -----
.PHONY: pages-preview
pages-preview: build-frontend ## Serve ./docs locally exactly like Pages would
	cd $(BUILD_DIR) && python3 -m http.server 4173

# ----- docker -----
.PHONY: docker-build
docker-build: ## Build amd64 image
	docker buildx build --platform linux/amd64 -t $(GHCR_REPO):$(VERSION) -t $(GHCR_REPO):latest --load $(BACKEND_DIR)

.PHONY: docker-push
docker-push: ## Push amd64 image to GHCR
	docker buildx build --platform linux/amd64 -t $(GHCR_REPO):$(VERSION) -t $(GHCR_REPO):latest --push $(BACKEND_DIR)

.PHONY: release
release: ## Tag + build & push image
	@test -n "$(V)" || (echo "usage: make release V=vX.Y.Z" && exit 1)
	git tag -a $(V) -m "release $(V)"
	git push origin $(V)
	$(MAKE) docker-push VERSION=$(V)

# ----- compose -----
.PHONY: compose-up
compose-up: ## Start local prod-mirror stack
	cd $(DEPLOY_DIR) && docker compose up -d

.PHONY: compose-dev-up
compose-dev-up: ## Start dev stack (builds locally)
	cd $(DEPLOY_DIR) && docker compose -f docker-compose.yml -f docker-compose.dev.yml up

.PHONY: compose-down
compose-down: ## Stop local stack
	cd $(DEPLOY_DIR) && docker compose down

# ----- hooks runnable manually -----
.PHONY: hooks-pre-commit hooks-commit-msg hooks-pre-push
hooks-pre-commit: ## Run pre-commit hook manually
	.githooks/pre-commit
hooks-commit-msg: ## Validate a commit message file ($(MSG))
	.githooks/commit-msg $(MSG)
hooks-pre-push: ## Run pre-push hook manually
	.githooks/pre-push

# ----- clean -----
.PHONY: clean
clean: ## Remove build artefacts
	rm -rf $(BACKEND_DIR)/bin $(FRONTEND_DIR)/dist $(FRONTEND_DIR)/node_modules $(BACKEND_DIR)/coverage.out
	@echo "cleaned. (docs/ left intact — it's the Pages publish dir)"

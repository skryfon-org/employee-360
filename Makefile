SHELL       := /bin/bash
BACKEND_DIR := backend
SCOPE       ?= employee-360

.PHONY: help bootstrap dev dev-api worker dev-admin dev-employee \
        migrate migrate-down migrate-status migrate-version migrate-reset \
        seed bootstrap-admin \
        test test-all test-backend test-backend-integration test-backend-cover cover-func cover-html test-clients \
        check check-backend check-clients check-structure fmt-check fmt \
        lint lint-backend lint-clients typecheck \
        build build-backend build-clients build-bin vet tidy clean \
        swagger generate images image-api image-admin image-employee

help: ## Show this help menu
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## --- Bootstrap ---

bootstrap: ## Install frontend and backend dependencies
	@if [ -f package.json ]; then pnpm install; fi
	cd $(BACKEND_DIR) && go mod download

## --- Dev ---

dev: ## Run backend API + worker + admin + employee clients concurrently
	@$(MAKE) -j4 dev-api worker dev-admin dev-employee

dev-api: ## Run the Go backend API (hot reload via air if installed, else go run)
	@if command -v air >/dev/null 2>&1; then \
		cd $(BACKEND_DIR) && air; \
	else \
		echo "air not installed (go install github.com/air-verse/air@latest) — falling back to go run"; \
		cd $(BACKEND_DIR) && go run ./cmd/api; \
	fi

worker: ## Run the River background worker (email outbox); run alongside `make dev`
	cd $(BACKEND_DIR) && go run ./cmd/worker

dev-admin: ## Run admin client (clients/admin)
	pnpm --filter admin dev

dev-employee: ## Run employee client (clients/employee)
	pnpm --filter employee dev

## --- Migrations ---

migrate: ## Apply all pending database migrations
	cd $(BACKEND_DIR) && go run ./cmd/migrate up

migrate-down: ## Roll back the most recent database migration
	cd $(BACKEND_DIR) && go run ./cmd/migrate down

migrate-status: ## Show migration status
	cd $(BACKEND_DIR) && go run ./cmd/migrate status

migrate-version: ## Show current database migration version
	cd $(BACKEND_DIR) && go run ./cmd/migrate version

migrate-reset: ## Reset database migrations (revert all and reapply)
	cd $(BACKEND_DIR) && go run ./cmd/migrate reset

## --- Seeding & Bootstrap ---

bootstrap-admin: ## Seed core roles, tenant, and Super Admin
	cd $(BACKEND_DIR) && go run ./cmd/bootstrap

seed: bootstrap-admin ## Run all dev seeding

## --- Tests ---

test-all: check ## Run lint, typecheck, and all test suites

test: test-backend test-clients ## Run backend and frontend tests

test-backend: ## Run Go unit tests (excludes integration-tagged tests, no DB required)
	cd $(BACKEND_DIR) && go test ./...

# -p 1 runs packages serially: they share the one <DATABASE_NAME>_test database
# (created and migrated on first use), and several tests wipe or reuse its rows.
test-backend-integration: ## Run Go integration tests (needs live PostgreSQL, CREATEDB privilege and access to the "postgres" maintenance DB; creates and migrates <DATABASE_NAME>_test)
	cd $(BACKEND_DIR) && go test -tags=integration -p 1 ./...

test-backend-cover: ## Run Go tests with coverage output
	cd $(BACKEND_DIR) && go test -coverprofile=coverage.out ./...

cover-func: ## Show Go test coverage breakdown by function
	cd $(BACKEND_DIR) && go tool cover -func=coverage.out

cover-html: ## Open Go test coverage report in browser
	cd $(BACKEND_DIR) && go tool cover -html=coverage.out

test-clients: ## Run frontend tests
	@if [ -f package.json ]; then pnpm -r test; else echo "Frontend clients not initialized yet (Cycle 1 scope)"; fi

## --- Quality & Linting ---

check: check-backend ## Run full verification suite (backend + frontend)
	@if [ -f package.json ]; then pnpm -r lint && pnpm -r typecheck; fi

check-backend: build-backend vet fmt-check check-structure test-backend test-backend-integration ## Run the backend-only verification suite (used by CI; requires a live PostgreSQL for integration tests)

check-clients: lint-clients typecheck test-clients build-clients ## Run the frontend-only verification suite (used by CI)

fmt-check: ## Fail if any backend file needs gofmt formatting
	@unformatted="$$(cd $(BACKEND_DIR) && gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needs to be run on:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

fmt: ## Auto-format backend Go files with gofmt
	cd $(BACKEND_DIR) && gofmt -w .

check-structure: ## Verify required backend scaffolding directories exist
	@missing=0; \
	for d in internal/usecase/interface internal/usecase/implementation internal/usecase/implementation/ucshared; do \
		if [ ! -d "$(BACKEND_DIR)/$$d" ]; then \
			echo "missing required directory: $(BACKEND_DIR)/$$d"; \
			missing=1; \
		fi; \
	done; \
	exit $$missing

lint: lint-backend lint-clients ## Run all linters

lint-backend: ## Run golangci-lint on backend (falls back to go vet if not installed)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		cd $(BACKEND_DIR) && golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, falling back to go vet"; \
		cd $(BACKEND_DIR) && go vet ./...; \
	fi

lint-clients: ## Run ESLint on frontend clients
	@if [ -f package.json ]; then pnpm -r lint; else echo "Frontend clients not initialized yet (Cycle 1 scope)"; fi

typecheck: ## Run TypeScript type checking
	@if [ -f package.json ]; then pnpm -r typecheck; else echo "Frontend clients not initialized yet (Cycle 1 scope)"; fi

vet: ## Run go vet on backend
	cd $(BACKEND_DIR) && go vet ./...

tidy: ## Tidy Go module dependencies
	cd $(BACKEND_DIR) && go mod tidy

## --- Build & Clean ---

build: build-backend ## Compile backend and build client bundles
	@if [ -f package.json ]; then pnpm -r build; fi

build-backend: ## Compile every backend package (matches "go build ./..." acceptance criterion)
	cd $(BACKEND_DIR) && go build ./...

build-clients: ## Build frontend client bundles
	@if [ -f package.json ]; then pnpm -r build; else echo "Frontend clients not initialized yet (Cycle 1 scope)"; fi

build-bin: ## Build backend binaries into backend/bin
	cd $(BACKEND_DIR) && go build -o bin/api ./cmd/api
	cd $(BACKEND_DIR) && go build -o bin/migrate ./cmd/migrate
	cd $(BACKEND_DIR) && go build -o bin/bootstrap ./cmd/bootstrap
	cd $(BACKEND_DIR) && go build -o bin/worker ./cmd/worker

clean: ## Remove build artifacts and temporary files
	rm -rf $(BACKEND_DIR)/bin $(BACKEND_DIR)/tmp $(BACKEND_DIR)/coverage.out
	@if [ -f package.json ]; then pnpm -r exec rm -rf dist .turbo; fi

## --- Container images (Podman; deployment: docs/self-hosting.md) ---

IMAGE_TAG ?= dev
VITE_API_BASE_URL ?= http://localhost:8080
VITE_EMPLOYEE_APP_URL ?= http://localhost:5174

images: image-api image-admin image-employee ## Build all deployable images with Podman

image-api: ## Build the API/worker/migrate image
	podman build -f infra/Containerfile.api -t localhost/employee360-api:$(IMAGE_TAG) .

image-admin: ## Build the admin SPA image
	podman build -f infra/Containerfile.web --build-arg APP=admin \
		--build-arg VITE_API_BASE_URL=$(VITE_API_BASE_URL) \
		--build-arg VITE_EMPLOYEE_APP_URL=$(VITE_EMPLOYEE_APP_URL) \
		-t localhost/employee360-admin:$(IMAGE_TAG) .

image-employee: ## Build the employee SPA image
	podman build -f infra/Containerfile.web --build-arg APP=employee \
		--build-arg VITE_API_BASE_URL=$(VITE_API_BASE_URL) \
		-t localhost/employee360-employee:$(IMAGE_TAG) .

## --- API Docs ---

swagger: ## Regenerate Swagger/OpenAPI docs from annotations (run after changing @-annotations)
	@cd $(BACKEND_DIR) && go run github.com/swaggo/swag/cmd/swag@v1.16.6 init \
		-g cmd/api/main.go \
		-o docs \
		--parseDependency \
		--parseInternal

generate: swagger ## Regenerate backend Swagger/OpenAPI docs and the frontend Orval API client
	@if [ -f package.json ]; then pnpm generate:api; fi

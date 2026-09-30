# Local development helpers. Run `make help` for the list.
.DEFAULT_GOAL := help
SHELL := /bin/bash

.PHONY: help install dev dev-api dev-web test test-api test-web lint lint-api lint-web build build-api build-web clean up down logs e2e

help: ## Show this help
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-12s %s\n", $$1, $$2}'

install: ## Install frontend dependencies from the lockfile
	cd frontend && npm ci

dev: ## Run the API and the UI together (Ctrl+C stops both)
	@trap 'kill 0' EXIT; $(MAKE) dev-api & $(MAKE) dev-web & wait

dev-api: ## Run the Go API on :8080
	cd backend && go run ./cmd/api

dev-web: ## Run the Vite dev server on :5173 (proxies /api to :8080)
	cd frontend && npm run dev

test: test-api test-web ## Run all unit tests

test-api: ## Go tests with the race detector
	cd backend && go test -race -cover ./...

test-web: ## Vitest
	cd frontend && npm test

lint: lint-api lint-web ## Lint and typecheck everything

GOLANGCI_LINT_VERSION ?= v2.14.0

lint-api: ## gofmt check, go vet and golangci-lint (same version as CI)
	cd backend && test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }
	cd backend && go vet ./...
	cd backend && go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

lint-web: ## oxlint and tsc
	cd frontend && npm run lint && npm run typecheck

build: build-api build-web ## Build the API binary and the UI bundle

build-api: ## Static, stripped Go binary in backend/bin/
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o bin/api ./cmd/api

build-web: ## Production UI bundle in frontend/dist/
	cd frontend && npm run build

up: ## Build and start the container stack on http://localhost:8080
	docker compose up -d --build --wait

down: ## Stop the container stack
	docker compose down

e2e: ## Playwright end-to-end tests against the running stack (make up)
	cd e2e && npm ci && npx playwright test

logs: ## Follow container logs
	docker compose logs -f

clean: ## Remove build output
	rm -rf backend/bin frontend/dist

# ---- Defaults & Help --------------------------------------------------------
SHELL := /bin/bash
.DEFAULT_GOAL := help

.PHONY: help
help: ## Display this help screen
	@grep -h -E '^[a-zA-Z0-9/_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-28s\033[0m %s\n", $$1, $$2}'

# ---- Project settings -------------------------------------------------------
APP       ?= pg_watcher
BIN       ?= bin
MAIN      ?= cmd/pg_watcher/main.go
RELEASE   ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
LDFLAGS   := -s -w -X main.build=$(RELEASE)
CGO_ENABLED ?= 0
VERBOSE     ?=
DOCKER_IMAGE ?= telegraf-pgwatcher
GOLANGCI_LINT ?= $(shell which golangci-lint 2>/dev/null || echo $(shell go env GOPATH)/bin/golangci-lint)

# ---- Tasks ------------------------------------------------------------------
.PHONY: all build docker_build test test_pg_watcher test_telegraf test_all lint clean vars run tidy

all: lint test_all build ## Run linter, all tests, then build

build: ## Build package
	@mkdir -p "$(BIN)"
	@echo "==> building $(BIN)/$(APP) from $(MAIN) (version: $(RELEASE))"
	@CGO_ENABLED=$(CGO_ENABLED) go build $(if $(VERBOSE),-v,) -ldflags "$(LDFLAGS)" -o "$(BIN)/$(APP)" "$(MAIN)"
	@echo "==> ok: $(BIN)/$(APP)"

docker_build: ## Build Docker image with telegraf and pg_watcher
	@echo "==> building Docker image $(DOCKER_IMAGE):latest (version: $(RELEASE))"
	@docker build --build-arg VERSION=$(RELEASE) -f docker/telegraf-pg_watcher/Dockerfile -t $(DOCKER_IMAGE):latest .
	@echo "==> ok: $(DOCKER_IMAGE):latest"

test: ## Run unit tests
	@echo "==> unit tests"
	@go test -race -coverprofile=coverage.out ./...
	@echo "==> coverage written to coverage.out"

test_pg_watcher: build ## Test pg_watcher only (build + PostgreSQL + pg_watcher)
	@echo "==> integration tests"
	@echo "==> cleaning up old containers"
	@cd docker && docker compose -f docker-compose-pg.yml down -v 2>/dev/null || true
	@echo "==> starting PostgreSQL container"
	@cd docker && docker compose -f docker-compose-pg.yml up -d
	@echo "==> waiting for PostgreSQL to initialize"
	@for i in {1..30}; do \
		if psql "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=postgres" -c "SELECT 1" >/dev/null 2>&1; then \
			echo "==> PostgreSQL ready on 127.0.0.1:5432"; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then \
			echo "==> ERROR: PostgreSQL not ready after 30 seconds"; \
			cd docker && docker compose -f docker-compose-pg.yml down -v; \
			exit 1; \
		fi; \
		sleep 1; \
	done
	@echo "==> testing pg_watcher"
	@if ./$(BIN)/$(APP) \
		-db-name=postgres \
		-j 1 \
		-conn="user=postgres password=postgres host=127.0.0.1 port=5432 sslmode=disable" \
		-sql-cmd="SELECT datname, datconnlimit FROM pg_database where datname='postgres'"; then \
		echo "==> pg_watcher test passed"; \
	else \
		echo "==> ERROR: pg_watcher test failed"; \
		cd docker && docker compose -f docker-compose-pg.yml down -v; \
		exit 1; \
	fi
	@echo "==> stopping PostgreSQL container"
	@cd docker && docker compose -f docker-compose-pg.yml down -v
	@echo "==> integration tests passed"

test_telegraf: ## Test full stack (PostgreSQL + Telegraf + pg_watcher)
	@echo "==> full stack integration test"
	@echo "==> cleaning up old containers"
	@cd docker && docker compose -f docker-compose-telegraf.yml down -v 2>/dev/null || true
	@echo "==> starting PostgreSQL + Telegraf stack"
	@cd docker && docker compose -f docker-compose-telegraf.yml up -d
	@echo "==> waiting for PostgreSQL to be healthy"
	@for i in {1..30}; do \
		if psql "host=127.0.0.1 port=5432 user=postgres password=postgres dbname=testdb" -c "SELECT 1" >/dev/null 2>&1; then \
			echo "==> PostgreSQL ready on 127.0.0.1:5432"; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then \
			echo "==> ERROR: PostgreSQL not ready after 30 seconds"; \
			cd docker && docker compose -f docker-compose-telegraf.yml down -v; \
			exit 1; \
		fi; \
		sleep 1; \
	done
	@echo "==> waiting for Telegraf to collect metrics (35 seconds)"
	@sleep 35
	@echo "==> checking Telegraf logs for metrics"
	@if docker logs pg_watcher_telegraf --tail 200 | grep "tlf_" >/dev/null 2>&1; then \
		echo "==> metrics found!"; \
		echo ""; \
		echo "==> Sample metrics collected:"; \
		docker logs pg_watcher_telegraf --tail 200 | grep "tlf_" | head -20; \
		echo ""; \
	else \
		echo "==> ERROR: no metrics found"; \
		docker logs pg_watcher_telegraf --tail 50; \
		cd docker && docker compose -f docker-compose-telegraf.yml down -v; \
		exit 1; \
	fi
	@echo "==> stopping containers"
	@cd docker && docker compose -f docker-compose-telegraf.yml down -v
	@echo "==> full stack test passed"

test_all: test test_pg_watcher test_telegraf ## Run all tests (unit + pg_watcher + telegraf)

lint: ## Run golangci-lint
	@echo "==> lint"
	@if ! command -v golangci-lint &>/dev/null && [ ! -f "$(GOLANGCI_LINT)" ]; then \
		echo "==> installing golangci-lint"; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin; \
	fi
	@$(GOLANGCI_LINT) run -c ./.golangci.yml --timeout 3m ./...

clean: ## Clean build artifacts
	@echo "==> clean"
	@rm -rf "$(BIN)" coverage.out

run: build ## Build & run
	@./$(BIN)/$(APP) -version

tidy: ## go mod tidy
	@go mod tidy -v

vars: ## Print useful vars (debug)
	@echo "APP      = $(APP)"
	@echo "BIN      = $(BIN)"
	@echo "MAIN     = $(MAIN)"
	@echo "RELEASE  = $(RELEASE)"
	@echo "LDFLAGS  = $(LDFLAGS)"
	@echo "CGO      = $(CGO_ENABLED)"


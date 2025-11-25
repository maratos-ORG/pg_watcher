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

# ---- Tasks ------------------------------------------------------------------
.PHONY: all build install test lint clean vars run tidy vagrant_up vagrant_destroy

all: lint test build ## Run linter, tests, then build

build: ## Build package
	@mkdir -p "$(BIN)"
	@echo "==> building $(BIN)/$(APP) from $(MAIN) (version: $(RELEASE))"
	@CGO_ENABLED=$(CGO_ENABLED) go build $(if $(VERBOSE),-v,) -ldflags "$(LDFLAGS)" -o "$(BIN)/$(APP)" "$(MAIN)"
	@echo "==> ok: $(BIN)/$(APP)"

install: ## go install into GOBIN/GOPATH/bin
	@echo "==> go install (version: $(RELEASE))"
	@CGO_ENABLED=$(CGO_ENABLED) go install -ldflags "$(LDFLAGS)" ./cmd/pg_watcher

test: ## Run tests
	@echo "==> tests"
	@go test -race -coverprofile=coverage.out ./...
	@echo "==> coverage written to coverage.out"

vagrant_up:
	@echo "Arch: $(ARCH) -> using $(VAGRANTFILE)"
	@cd Vagrant/PostgresDB && \
		$(VAGRANT_ENV) vagrant up && \
		$(VAGRANT_ENV) vagrant provision && \

vagrant_destroy:
	@ARCH=$$(uname -m); \
	if [[ "$$ARCH" == "arm64" ]]; then \
		export VAGRANT_VAGRANTFILE="Vagrantfile_MAC_ARM"; \
	else \
		export VAGRANT_VAGRANTFILE="Vagrantfile_MAC_INTEL"; \
	fi; \
	cd Vagrant/PostgresDB && \
	vagrant destroy -f

lint: ## Run golangci-lint
	@echo "==> lint"
	@golangci-lint run -c ./.golangci.yml --timeout 3m ./...

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
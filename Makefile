.DEFAULT_GOAL := help

# Makefile

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint

## Tool Versions
# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.13.2

# Default tag prefix. Override with VERSION_PREFIX= if you do not want one.
VERSION_PREFIX ?= v

# renovate: datasource=github-releases depName=gi8lino/dev-tools
DEV_TOOLS_VERSION ?= v0.7.0

include bin/dev-tools.mk
include $(call dev-tools-module,tag)
include $(call dev-tools-module,help)

.PHONY: tag
tag: current ## Show the current semantic version tag.

##@ Development

.PHONY: download
download: ## Download go packages
	go mod download

KARMA_ARGS ?= --help

.PHONY: run
run: ## Run karma. Override KARMA_ARGS to pass command-line arguments.
	go run ./cmd/karma $(KARMA_ARGS)

.PHONY: fmt
fmt: ## Format Go source files.
	go fmt ./...

.PHONY: fmt-check
fmt-check: ## Fail when Go source files need formatting.
	@files="$$(gofmt -l .)"; 	if [ -n "$$files" ]; then 		echo "Go files need formatting:"; 		echo "$$files"; 		exit 1; 	fi

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt-check vet ## Run unit tests without modifying the working tree.
	go test -covermode=atomic -count=1 -parallel=4 -timeout=5m ./...

.PHONY: cover
cover: ## Display test coverage
	go test -coverprofile=coverage.out -covermode=atomic -count=1 -parallel=4 -timeout=5m ./...
	go tool cover -html=coverage.out

.PHONY: clean
clean: ## Clean up generated files
	rm -f coverage.out coverage.html

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter.
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes.
	$(GOLANGCI_LINT) run --fix

##@ Dependencies

.PHONY: golangci-lint
golangci-lint: $(GO_INSTALL_TOOL) | $(LOCALBIN) ## Download golangci-lint locally if necessary.
	@$(GO_INSTALL_TOOL) \
		--target "$(GOLANGCI_LINT)" \
		--package github.com/golangci/golangci-lint/v2/cmd/golangci-lint \
		--tool-version "$(GOLANGCI_LINT_VERSION)"

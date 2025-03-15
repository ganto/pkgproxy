NAME    := $(shell basename `pwd`)
SOURCE  := $(shell find . -name "*.go")
VERSION := $(shell git describe --always)

BIN_DIR := bin/

GO_INSTALL_ARGS       :=
GO_INSTALL_ARGS_EXTRA :=
# Build: https://golang.org/cmd/go/#hdr-Compile_packages_and_dependencies
GO_BUILD_ARGS         := -a -v -trimpath
GO_BUILD_ARGS_EXTRA   :=

# Vet: https://pkg.go.dev/cmd/vet
GO_VET_ARGS           :=

GO_TEST_ARGS          := -v -race
GO_TEST_ARGS_EXTRA    :=

# By default build static binaries
CGO_ENABLED           := 0

export GO111MODULE=on

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

## Format sources files with goimports
.PHONY: format
format: ## Format source files with `goimports`
	$(info ****************************************************)
	$(info ********** EXECUTING 'format' MAKE TARGET **********)
	$(info ****************************************************)
	@command -v goimports 2>&1 >/dev/null || go install $(GO_INSTALL_ARGS) $(GO_INSTALL_ARGS_EXTRA) golang.org/x/tools/cmd/goimports@latest
	goimports -w $(SOURCE)

.PHONY: vet
vet: ## Run go vet against code
	$(info *************************************************)
	$(info ********** EXECUTING 'vet' MAKE TARGET **********)
	$(info *************************************************)
	go vet $(GO_VET_ARGS) ./...

.PHONY: generate
generate: ## Run code generation (if required)
	$(info ******************************************************)
	$(info ********** EXECUTING 'generate' MAKE TARGET **********)
	$(info ******************************************************)
	go generate -v ./...

.PHONY: ci-lint
ci-lint: ## Run all lint related tests against the codebase (will use the .golangci.yml config)
	$(info *****************************************************)
	$(info ********** EXECUTING 'ci-lint' MAKE TARGET **********)
	$(info *****************************************************)
	@command -v golangci-lint 2>&1 >/dev/null || go install $(GO_INSTALL_ARGS) $(GO_INSTALL_ARGS_EXTRA) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint -v run

.PHONY: ci-check
ci-check: ci-lint govulncheck test ## Run various check commands intended for CI use (lint, govulncheck, test, ...)

.PHONY: lint
lint: ci-lint ## Alias for ci-lint

.PHONY: lint-fix
lint-fix: ci-lint ## Run golangci-lint linter and perform fixes
	$(info ******************************************************)
	$(info ********** EXECUTING 'lint-fix' MAKE TARGET **********)
	$(info ******************************************************)
	golangci-lint -v run --fix

.PHONY: test
test: ## Run the tests against the codebase
	$(info **************************************************)
	$(info ********** EXECUTING 'test' MAKE TARGET **********)
	$(info **************************************************)
	go test $(GO_TEST_ARGS) $(GO_TEST_ARGS_EXTRA) ./...

.PHONY: coverage
coverage: ## Generates test coverage report
	$(info ******************************************************)
	$(info ********** EXECUTING 'coverage' MAKE TARGET **********)
	$(info ******************************************************)
	rm -f coverage.out
	go test ./... -coverpkg=./... -coverprofile=coverage.out

.PHONY: govulncheck
govulncheck: ## Run Go vulnerability check
	$(info *********************************************************)
	$(info ********** EXECUTING 'govulncheck' MAKE TARGET **********)
	$(info *********************************************************)
	@command -v govulncheck 2>&1 >/dev/null || go install $(GO_INSTALL_ARGS) $(GO_INSTALL_ARGS_EXTRA) golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck -show verbose ./... || true

##@ Build

.PHONY: build
build: generate ci-build ## Build the application binary

.PHONY: ci-build
ci-build: ## To be called to build the application binary in a CI pipeline
	$(info ******************************************************)
	$(info ********** EXECUTING 'ci-build' MAKE TARGET **********)
	$(info ******************************************************)
	CGO_ENABLED=$(CGO_ENABLED) go build $(GO_BUILD_ARGS) $(GO_BUILD_ARGS_EXTRA) -o $(BIN_DIR)$(NAME) .

.PHONY: run
run: format vet generate ## Run the application from your host
	$(info *************************************************)
	$(info ********** EXECUTING 'run' MAKE TARGET **********)
	$(info *************************************************)
	PKGPROXY_CONFIG=./configs/pkgproxy.yaml CGO_ENABLED=$(CGO_ENABLED) go run . serve --host 0.0.0.0 --debug

.PHONY: clean
clean: ## Cleanup binary
	$(info ***************************************************)
	$(info ********** EXECUTING 'clean' MAKE TARGET **********)
	$(info ***************************************************)
	rm -rvf $(BIN_DIR)

.PHONY: all
all: format vet lint govulncheck test build

.DEFAULT_GOAL := help
MAKEFLAGS += --silent --no-print-directory

BIN_DIR := ./bin
SCRIPTS_DIR := ./scripts
APP_NAME := nobl9-openslo
LDFLAGS += -s -w

# renovate datasource=github-releases depName=securego/gosec
GOSEC_VERSION := v2.22.7
# renovate datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION := v1.64.8
# renovate datasource=go depName=golang.org/x/vuln/cmd/govulncheck
GOVULNCHECK_VERSION := v1.1.4
# renovate datasource=go depName=golang.org/x/tools/cmd/goimports
GOIMPORTS_VERSION := v0.35.0

# Check if the program is present in $PATH and install otherwise.
# ${1} - oneOf{binary,yarn}
# ${2} - program name
define _ensure_installed
	LOCAL_BIN_DIR=$(BIN_DIR) $(SCRIPTS_DIR)/ensure_installed.sh "${1}" "${2}"
endef

# Install Go binary using 'go install' with an output directory set via $GOBIN.
# ${1} - repository url
define _install_go_binary
	GOBIN=$(realpath $(BIN_DIR)) go install "${1}"
endef

# Print Makefile target step description for check.
# Only print 'check' steps this way, and not dependent steps, like 'install'.
# ${1} - step description
define _print_check_step
	printf -- '------\n%s...\n' "${1}"
endef

.PHONY: build
## Build nobl9-openslo binary.
build:
	go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) ./cmd

.PHONY: test
## Run all unit tests.
test:
	go test -race -cover ./...

.PHONY: check check/vet check/lint check/gosec check/spell check/trailing check/markdown check/format check/generate check/vulns
## Run all checks.
check: check/vet check/lint check/gosec check/spell check/trailing check/markdown check/format check/generate check/vulns

## Run 'go vet' on the whole project.
check/vet:
	$(call _print_check_step,Running go vet)
	go vet ./...

## Run golangci-lint all-in-one linter with configuration defined inside .golangci.yml.
check/lint:
	$(call _print_check_step,Running golangci-lint)
	$(call _ensure_installed,binary,golangci-lint)
	$(BIN_DIR)/golangci-lint run

## Check for security problems using gosec, which inspects the Go code by scanning the AST.
check/gosec:
	$(call _print_check_step,Running gosec)
	$(call _ensure_installed,binary,gosec)
	$(BIN_DIR)/gosec -exclude-generated -quiet ./...

## Check spelling, rules are defined in cspell.json.
check/spell:
	$(call _print_check_step,Verifying spelling)
	$(call _ensure_installed,yarn,cspell)
	yarn --silent cspell --no-progress '**/**'

## Check for trailing white spaces in any of the projects' files.
check/trailing:
	$(call _print_check_step,Looking for trailing whitespaces)
	$(SCRIPTS_DIR)/check-trailing-whitespaces.bash

## Check markdown files for potential issues with markdownlint.
check/markdown:
	$(call _print_check_step,Verifying Markdown files)
	$(call _ensure_installed,yarn,markdownlint)
	yarn --silent markdownlint '**/*.md' --ignore node_modules

## Check for potential vulnerabilities across all Go dependencies.
check/vulns:
	$(call _print_check_step,Running govulncheck)
	$(call _ensure_installed,binary,govulncheck)
	$(BIN_DIR)/govulncheck ./...

## Verify if the auto generated code has been committed.
check/generate:
	$(call _print_check_step,Checking if generated code matches the provided definitions)
	$(SCRIPTS_DIR)/check-generate.sh

## Verify if the files are formatted.
## You must first commit the changes, otherwise it won't detect the diffs.
check/format:
	$(call _print_check_step,Checking if files are formatted)
	$(SCRIPTS_DIR)/check-formatting.sh

.PHONY: generate generate/code
## Auto generate files.
generate: generate/code

## Generate Golang code.
generate/code:
	echo "Generating Go code..."
	go generate ./...

.PHONY: format format/go format/cspell
## Format files.
format: format/go format/cspell

## Format Go files.
format/go:
	echo "Formatting Go files..."
	$(call _ensure_installed,binary,goimports)
	gofmt -l -w -s .
	$(BIN_DIR)/goimports -local=github.com/nobl9/nobl9-openslo -w .

## Format cspell config file.
format/cspell:
	echo "Formatting cspell.yaml configuration (words list)..."
	$(call _ensure_installed,yarn,yaml)
	yarn --silent format-cspell-config

.PHONY: install install/yarn install/golangci-lint install/gosec install/govulncheck install/goimports
## Install all dev dependencies.
install: install/yarn install/golangci-lint install/gosec install/govulncheck install/goimports

## Install JS dependencies with yarn.
install/yarn:
	echo "Installing yarn dependencies..."
	yarn --silent install

## Install golangci-lint (https://golangci-lint.run).
install/golangci-lint:
	echo "Installing golangci-lint..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh |\
 		sh -s -- -b $(BIN_DIR) $(GOLANGCI_LINT_VERSION)

## Install gosec (https://github.com/securego/gosec).
install/gosec:
	echo "Installing gosec..."
	curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh |\
 		sh -s -- -b $(BIN_DIR) $(GOSEC_VERSION)

## Install govulncheck (https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck).
install/govulncheck:
	echo "Installing govulncheck..."
	$(call _install_go_binary,golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION))

## Install goimports (https://pkg.go.dev/golang.org/x/tools/cmd/goimports).
install/goimports:
	echo "Installing goimports..."
	$(call _install_go_binary,golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION))

.PHONY: help
## Print this help message.
help:
	$(SCRIPTS_DIR)/makefile-help.awk $(MAKEFILE_LIST)

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
GO_IMPORTS ?= $(LOCALBIN)/goimports
GCI ?= $(LOCALBIN)/gci

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: go_fmt
go_fmt: ## Run go fmt against code.
	gofmt -w -s .

.PHONY: fmt_imports
fmt_imports: $(GCI) ## Run goimports against code.
	$(GCI) write ./ --skip-generated -s standard -s default -s 'prefix(github.com/qdrant)'

.PHONY: fmt
format: go_fmt fmt_imports ## Format the code.

fmt: format ## Format the code.

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

lint: $(GCI) ## Run linters.
	bash -c 'files=$$(gofmt -l .) && echo $$files && [ -z "$$files" ]'
	golangci-lint run

.PHONY: test_unit
test_unit: ## Run unit tests.
	go test ./... -coverprofile cover.out

$(GO_IMPORTS): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install golang.org/x/tools/cmd/goimports@latest

$(GCI): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/daixiang0/gci@latest



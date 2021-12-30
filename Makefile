# Copyright 2021 TriggerMesh Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

KREPO      = triggermesh
KREPO_DESC = TriggerMesh Open Source Components (sources, targets, filter, router, etc.)

BASE_DIR          ?= $(CURDIR)
OUTPUT_DIR        ?= $(BASE_DIR)/_output

# Dynamically generate the list of commands based on the directory name cited in the cmd directory
COMMANDS          := $(notdir $(wildcard cmd/*))

BIN_OUTPUT_DIR    ?= $(OUTPUT_DIR)
DOCS_OUTPUT_DIR   ?= $(OUTPUT_DIR)
TEST_OUTPUT_DIR   ?= $(OUTPUT_DIR)
COVER_OUTPUT_DIR  ?= $(OUTPUT_DIR)
DIST_DIR          ?= $(OUTPUT_DIR)

# Rely on ko for building/publishing images and generating/deploying manifests
KO                ?= ko
KOFLAGS           ?=
IMAGE_TAG         ?= $(shell git rev-parse HEAD)

# Go build variables
GO                ?= go
GOFMT             ?= gofmt
GOLINT            ?= golangci-lint run --timeout 5m
GOTOOL            ?= go tool
GOTEST            ?= gotestsum --junitfile $(TEST_OUTPUT_DIR)/$(KREPO)-unit-tests.xml --format pkgname-and-test-fails --

GOPKGS             = ./cmd/... ./pkg/apis/... ./pkg/function/... ./pkg/routing/... ./pkg/sources/... ./pkg/targets/... ./pkg/transformation/...
LDFLAGS            = -w -s
LDFLAGS_STATIC     = $(LDFLAGS) -extldflags=-static

HAS_GOTESTSUM     := $(shell command -v gotestsum;)
HAS_GOLANGCI_LINT := $(shell command -v golangci-lint;)

.PHONY: help all build release vm-images test lint fmt fmt-test images clean install-gotestsum install-golangci-lint deploy undeploy

.DEFAULT_GOAL := build

all: codegen build test lint

# Verify lint and tests
install-gotestsum:
ifndef HAS_GOTESTSUM
	curl -SL https://github.com/gotestyourself/gotestsum/releases/download/v1.7.0/gotestsum_1.7.0_linux_amd64.tar.gz | tar -C $(shell go env GOPATH)/bin -zxf -
endif

install-golangci-lint:
ifndef HAS_GOLANGCI_LINT
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.41.1
endif

help: ## Display this help
	@awk 'BEGIN {FS = ":.*?## "; printf "\n$(KREPO_DESC)\n\nUsage:\n  make \033[36m<cmd>\033[0m\n"} /^[a-zA-Z0-9._-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: $(COMMANDS)  ## Build all artifacts

$(filter-out confluenttarget-adapter xslttransform-adapter, $(COMMANDS)): ## Build artifact
	$(GO) build -ldflags "$(LDFLAGS_STATIC)" -o $(BIN_OUTPUT_DIR)/$@ ./cmd/$@

confluenttarget-adapter:
	CGO_ENABLED=1 $(GO) build -ldflags "$(LDFLAGS_STATIC)" -o $(BIN_OUTPUT_DIR)/$@ ./cmd/$@

# Not statically linked
xslttransform-adapter: ## Builds XML related functionality
	CGO_ENABLED=1 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_OUTPUT_DIR)/$@ ./cmd/$@

deploy: ## Deploy TriggerMesh stack to default Kubernetes cluster using ko
	$(KO) apply -f $(BASE_DIR)/config

undeploy: ## Remove TriggerMesh stack from default Kubernetes cluster using ko
	$(KO) delete -f $(BASE_DIR)/config

vm-images:
	@$(MAKE) -C packer/

release: ## Publish container images and generate release manifests
	@mkdir -p $(DIST_DIR)
	$(KO) resolve -f config/ -l 'triggermesh.io/crd-install' > $(DIST_DIR)/triggermesh-crds.yaml
	@cp config/namespace/100-namespace.yaml $(DIST_DIR)/triggermesh.yaml
	$(KO) resolve $(KOFLAGS) -B -t latest -f config/ -l '!triggermesh.io/crd-install' > /dev/null
	$(KO) resolve $(KOFLAGS) -B -t $(IMAGE_TAG) --tag-only -f config/ -l '!triggermesh.io/crd-install' >> $(DIST_DIR)/triggermesh.yaml

gen-apidocs: ## Generate API docs
	GOPATH="" OUTPUT_DIR=$(DOCS_OUTPUT_DIR) ./hack/gen-api-reference-docs.sh

test: install-gotestsum ## Run unit tests
	@mkdir -p $(TEST_OUTPUT_DIR)
	$(GOTEST) -p=1 -race -cover -coverprofile=$(TEST_OUTPUT_DIR)/$(KREPO)-c.out $(GOPKGS)

cover: test ## Generate code coverage
	@mkdir -p $(COVER_OUTPUT_DIR)
	$(GOTOOL) cover -html=$(TEST_OUTPUT_DIR)/$(KREPO)-c.out -o $(COVER_OUTPUT_DIR)/$(KREPO)-coverage.html

lint: install-golangci-lint ## Lint source files
	$(GOLINT) $(GOPKGS)

fmt: ## Format source files
	$(GOFMT) -s -w $(shell $(GO) list -f '{{$$d := .Dir}}{{range .GoFiles}}{{$$d}}/{{.}} {{end}} {{$$d := .Dir}}{{range .TestGoFiles}}{{$$d}}/{{.}} {{end}}' $(GOPKGS))

fmt-test: ## Check source formatting
	@test -z $(shell $(GOFMT) -l $(shell $(GO) list -f '{{$$d := .Dir}}{{range .GoFiles}}{{$$d}}/{{.}} {{end}} {{$$d := .Dir}}{{range .TestGoFiles}}{{$$d}}/{{.}} {{end}}' $(GOPKGS)))

IMAGES = $(foreach cmd,$(COMMANDS),$(cmd).image)
images: $(IMAGES) ## Build container images
$(IMAGES): %.image:
	$(KO) publish --push=false -B --tag-only -t $(IMAGE_TAG) ./cmd/$*

clean: ## Clean build artifacts
	@for bin in $(COMMANDS) ; do \
		$(RM) -v $(BIN_OUTPUT_DIR)/$$bin; \
	done
	@$(RM) -v $(DIST_DIR)/triggermesh-crds.yaml $(DIST_DIR)/triggermesh.yaml
	@$(RM) -v $(TEST_OUTPUT_DIR)/$(KREPO)-c.out $(TEST_OUTPUT_DIR)/$(KREPO)-unit-tests.xml
	@$(RM) -v $(COVER_OUTPUT_DIR)/$(KREPO)-coverage.html

# Code generation
include $(BASE_DIR)/hack/inc.Codegen.mk

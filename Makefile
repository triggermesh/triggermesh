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
TARGETS           ?= linux/amd64

BIN_OUTPUT_DIR    ?= $(OUTPUT_DIR)
TEST_OUTPUT_DIR   ?= $(OUTPUT_DIR)
COVER_OUTPUT_DIR  ?= $(OUTPUT_DIR)
DIST_DIR          ?= $(OUTPUT_DIR)

# Docker build variables
DOCKER            ?= docker
IMAGE_REPO        ?= gcr.io/triggermesh
IMAGE_TAG         ?= latest
IMAGE_SHA         ?= $(shell git rev-parse HEAD)

# Rely on ko for dev style deployment
KO                ?= ko

KUBECTL           ?= kubectl
SED               ?= sed

# Go build variables
GO                ?= go
GOFMT             ?= gofmt
GOLINT            ?= golangci-lint run --timeout 5m
GOTOOL            ?= go tool
GOTEST            ?= gotestsum --junitfile $(TEST_OUTPUT_DIR)/$(KREPO)-unit-tests.xml --format pkgname-and-test-fails --

GOPKGS             = ./cmd/... ./pkg/apis/... ./pkg/function/... ./pkg/routing/... ./pkg/sources/... ./pkg/targets/... ./pkg/transformation/...
LDFLAGS            = -extldflags=-static -w -s

HAS_GOTESTSUM     := $(shell command -v gotestsum;)
HAS_GOLANGCI_LINT := $(shell command -v golangci-lint;)

.PHONY: help build install release test lint fmt fmt-test images cloudbuild-test cloudbuild clean install-gotestsum install-golangci-lint deploy undeploy

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

$(filter-out confluent-target-adapter, $(COMMANDS)): ## Build artifact
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_OUTPUT_DIR)/$@ ./cmd/$@

confluent-target-adapter:
	CGO_ENABLED=1 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_OUTPUT_DIR)/$@ ./cmd/$@

deploy: ## Deploy TriggerMesh stack to default Kubernetes cluster using ko
	CGO_ENABLED=1 $(KO) deploy -f $(BASE_DIR)/config

undeploy: ## Remove TriggerMesh stack from default Kubernetes cluster using ko
	$(KO) delete -f $(BASE_DIR)/config

release: ## Build release binaries
	@set -e ; \
	for bin in $(COMMANDS) ; do \
		for platform in $(TARGETS); do \
			GOOS=$${platform%/*} ; \
			GOARCH=$${platform#*/} ; \
			RELEASE_BINARY=$$bin-$${GOOS}-$${GOARCH} ; \
			CGO_ENABLED= ; \
			[ $${bin} == "confluent-target-adapter" ] && CGO_ENABLED=1 ; \
			[ $${GOOS} = "windows" ] && RELEASE_BINARY=$${RELEASE_BINARY}.exe ; \
			echo "GOOS=$${GOOS} GOARCH=$${GOARCH} $${CGO_ENABLED:+CGO_ENABLED=$${CGO_ENABLED}} $(GO) build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$${RELEASE_BINARY} ./cmd/$$bin" ; \
			GOOS=$${GOOS} GOARCH=$${GOARCH} $${CGO_ENABLED:+CGO_ENABLED=$${CGO_ENABLED}} $(GO) build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$${RELEASE_BINARY} ./cmd/$$bin ; \
		done ; \
	done

	$(KUBECTL) create -f config --dry-run=client -o yaml | \
	  $(SED) 's|ko://github.com/triggermesh/triggermesh/cmd/\(.*\)|$(IMAGE_REPO)/\1:${IMAGE_TAG}|' > $(DIST_DIR)/triggermesh.yaml

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
images: $(IMAGES) ## Builds container images
$(IMAGES): %.image:
	$(DOCKER) build -t $(IMAGE_REPO)/$* -f ./cmd/$*/Dockerfile .

CLOUDBUILD_TEST = $(foreach cmd,$(COMMANDS),$(cmd).cloudbuild-test)
cloudbuild-test: $(CLOUDBUILD_TEST) ## Test container image build with Google Cloud Build
$(CLOUDBUILD_TEST): %.cloudbuild-test:

# NOTE (antoineco): Cloud Build started failing recently with authentication errors when --no-push is specified.
# Pushing images with the "_" tag is our hack to avoid those errors and ensure the build cache is always updated.
	gcloud builds submit $(BASE_DIR) --config cloudbuild.yaml --substitutions _CMD=$*,COMMIT_SHA=${IMAGE_SHA},_KANIKO_IMAGE_TAG=_

CLOUDBUILD = $(foreach cmd,$(COMMANDS),$(cmd).cloudbuild)
cloudbuild: $(CLOUDBUILD) ## Build and publish image to GCR
$(CLOUDBUILD): %.cloudbuild:
	gcloud builds submit $(BASE_DIR) --config cloudbuild.yaml --substitutions _CMD=$*,COMMIT_SHA=${IMAGE_SHA},_KANIKO_IMAGE_TAG=${IMAGE_TAG}

clean: ## Clean build artifacts
	@for bin in $(COMMANDS) ; do \
		for platform in $(TARGETS); do \
			GOOS=$${platform%/*} ; \
			GOARCH=$${platform#*/} ; \
			RELEASE_BINARY=$$bin-$${GOOS}-$${GOARCH} ; \
			[ $${GOOS} = "windows" ] && RELEASE_BINARY=$${RELEASE_BINARY}.exe ; \
			$(RM) -v $(DIST_DIR)/$${RELEASE_BINARY}; \
		done ; \
		$(RM) -v $(BIN_OUTPUT_DIR)/$$bin; \
	done
	@$(RM) -v $(TEST_OUTPUT_DIR)/$(KREPO)-c.out $(TEST_OUTPUT_DIR)/$(KREPO)-unit-tests.xml
	@$(RM) -v $(COVER_OUTPUT_DIR)/$(KREPO)-coverage.html

# Code generation
include $(BASE_DIR)/hack/inc.Codegen.mk

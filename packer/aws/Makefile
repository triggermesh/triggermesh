TRIGGERMESH_VERSION ?= $(shell cat TRIGGERMESH_VERSION)

BASE_DIR   ?= $(CURDIR)
OUTPUT_DIR ?= $(BASE_DIR)/_output

.PHONY: build config.yaml

all: build

gen-config:
	@echo "Generating config.yaml..."
	@mkdir -p _output/
	@TRIGGERMESH_VERSION=$(TRIGGERMESH_VERSION) ./scripts/gen-config > $(OUTPUT_DIR)/config.yaml

build: gen-config
	@echo "Building AMI..."
	@TRIGGERMESH_VERSION=$(TRIGGERMESH_VERSION) K3OS_CONFIG=$(OUTPUT_DIR)/config.yaml packer build template.json

clean:
	@rm -rf $(OUTPUT_DIR)/config.yaml

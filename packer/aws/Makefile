BASE_DIR   ?= $(CURDIR)
OUTPUT_DIR ?= $(BASE_DIR)/_output

TRIGGERMESH_VERSION ?= $(shell git describe --tags --abbrev=0)
K3OS_CONFIG=$(OUTPUT_DIR)/config.yaml

export TRIGGERMESH_VERSION
export K3OS_CONFIG

.PHONY: build gen-config

all: build

gen-config:
	@echo "Generating config.yaml..."
	@mkdir -p _output/
	@./scripts/gen-config > $(K3OS_CONFIG)

# Requires AWS environment for Packer as per your preferred method:
# - https://www.packer.io/docs/builders/amazon/#static-credentials
# - https://www.packer.io/docs/builders/amazon/#environment-variables
# - https://www.packer.io/docs/builders/amazon/#shared-credentials-file
build: gen-config
	@echo "Building AMI..."
	@packer build template.json

clean:
	@rm -rvf $(K3OS_CONFIG)

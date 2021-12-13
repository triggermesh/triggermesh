DOCKER		?= docker
PLATFORM 	?= linux/amd64
GOALS		:= build tag push test

.PHONY: $(GOALS)

build:
	$(DOCKER) build -t $(IMAGE_TAG) -f Dockerfile $(CONTEXT) --platform $(PLATFORM)

tag:
	$(DOCKER) tag $(IMAGE_TAG) $(TAGS)

push:
	$(DOCKER) push --all-tags $(IMAGE_TAG)

test:

.SILENT:

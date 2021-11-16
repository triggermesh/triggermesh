.PHONY: test vet release

# would be great to make the bash location portable but not sure how
SHELL = /bin/bash -o pipefail

DIFFER := $(GOPATH)/bin/differ
WRITE_MAILMAP := $(GOPATH)/bin/write_mailmap
BUMP_VERSION := $(GOPATH)/bin/bump_version
MEGACHECK := $(GOPATH)/bin/megacheck

test: lint
	go test ./...

$(BUMP_VERSION):
	go get github.com/Shyp/bump_version

$(DIFFER):
	go get github.com/kevinburke/differ

$(MEGACHECK):
	go get honnef.co/go/tools/cmd/megacheck

$(WRITE_MAILMAP):
	go get github.com/kevinburke/write_mailmap

lint: | $(MEGACHECK)
	go vet ./...
	$(MEGACHECK) --ignore='github.com/kevinburke/twilio-go/*.go:S1002' ./...

race-test: vet
	go test -race ./...

race-test-short: vet
	go test -short -race ./...

release: race-test | $(DIFFER) $(BUMP_VERSION)
	$(DIFFER) $(MAKE) authors
	$(BUMP_VERSION) minor http.go

force: ;

AUTHORS.txt: force | $(WRITE_MAILMAP)
	$(WRITE_MAILMAP) > AUTHORS.txt

authors: AUTHORS.txt

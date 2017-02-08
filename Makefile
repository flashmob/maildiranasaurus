GIT ?= git
GO_VARS ?=
GO ?= go
COMMIT := $(shell $(GIT) rev-parse HEAD)
VERSION ?= $(shell $(GIT) describe --tags ${COMMIT} 2> /dev/null || echo "$(COMMIT)")
BUILD_TIME := $(shell LANG=en_US date +"%F_%T_%z")
ROOT := github.com/flashmob/maildiranasaurus
LD_FLAGS := -X $(ROOT).Version=$(VERSION) -X $(ROOT).Commit=$(COMMIT) -X $(ROOT).BuildTime=$(BUILD_TIME)

.PHONY: help clean dependencies test
help:
	@echo "Please use \`make <ROOT>' where <ROOT> is one of"
	@echo "  dependencies to go install the dependencies"
	@echo "  maildiranasaurus   to build the main binary for current platform"
	@echo "  test         to run unittests"

clean:
	rm -f maildiranasaurus

dependencies:
	$(GO_VARS) $(GO) list -f='{{ join .Deps "\n" }}' $(ROOT)/cmd/maildiranasaurus | grep -v $(ROOT) | tr '\n' ' ' | $(GO_VARS) xargs $(GO) get -u -v
	$(GO_VARS) $(GO) list -f='{{ join .Deps "\n" }}' $(ROOT)/cmd/maildiranasaurus | grep -v $(ROOT) | tr '\n' ' ' | $(GO_VARS) xargs $(GO) install -v

maildiranasaurus: *.go */*.go */*/*.go
	$(GO_VARS) $(GO) build -o="maildiranasaurus" -ldflags="$(LD_FLAGS)" $(ROOT)/cmd/maildiranasaurus

test: *.go */*.go */*/*.go
	$(GO_VARS) $(GO) test -v .


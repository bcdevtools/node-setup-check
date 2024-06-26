GIT_TAG := $(shell echo $(shell git describe --tags || git branch --show-current) | sed 's/^v//')
COMMIT  := $(shell git log -1 --format='%H')
BUILD_DATE	:= $(shell date '+%Y-%m-%d')

###############################################################################
###                                Build flags                              ###
###############################################################################

LD_FLAGS = -X github.com/bcdevtools/node-setup-check/constants.VERSION=$(GIT_TAG) \
            -X github.com/bcdevtools/node-setup-check/constants.COMMIT_HASH=$(COMMIT) \
            -X github.com/bcdevtools/node-setup-check/constants.BUILD_DATE=$(BUILD_DATE)

BUILD_FLAGS := -ldflags '$(LD_FLAGS)'

###############################################################################
###                                  Test                                   ###
###############################################################################

test: go.sum
	@echo "testing"
	@go test -v ./... -race -coverprofile=coverage.txt -covermode=atomic
.PHONY: test

###############################################################################
###                                  Build                                  ###
###############################################################################

build: go.sum
	@echo "Building nodesc binary..."
	@echo "Flags $(BUILD_FLAGS)"
	@go build -mod=readonly $(BUILD_FLAGS) -o build/nodesc ./cmd/nodesc
	@echo "Builded Node Setup Check successfully"
.PHONY: build

###############################################################################
###                                 Install                                 ###
###############################################################################

install: go.sum
	@echo "Installing nodesc binary..."
	@echo "Flags $(BUILD_FLAGS)"
	@go install -mod=readonly $(BUILD_FLAGS) ./cmd/nodesc
	@echo "Installed Node Setup Check successfully"
.PHONY: install
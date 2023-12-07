# Config for the binaries you want to build
NAME=coco
REPO=github.com/shaharia-lab/${NAME}
VERSION ?= "dev-$(shell git rev-parse HEAD --short)"

BINARY=$(NAME)
BINARY_SRC=$(REPO)

SRC_DIRS=pkg

# Build configuration
BUILD_DIR ?= $(CURDIR)/out
BUILD_GOOS ?= $(shell go env GOOS)
BUILD_GOARCH ?= $(shell go env GOARCH)
GO_LINKER_FLAGS=-ldflags="-s -w -X main.version=$(VERSION)"

# Build tweaks for windows
ifeq (${BUILD_GOOS}, windows)
BINARY := $(BINARY).exe
endif

# Other config
NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

.PHONY: all clean deps build

all: clean deps build

# Install dependencies
deps:
	@printf "$(OK_COLOR)==> Installing dependencies$(NO_COLOR)\n"
	@go mod vendor
	@CGO_ENABLED=0 go generate

# Builds the project
build:
	@mkdir -p ${BUILD_DIR}
	@printf "$(OK_COLOR)==> Building ${BINARY} for ${BUILD_GOOS}/${BUILD_GOARCH}: $(NO_COLOR)\n"
	CGO_ENABLED=0 GOARCH=${BUILD_GOARCH} GOOS=${BUILD_GOOS} \
	  go build -o ${BUILD_DIR}/${BINARY} ${GO_LINKER_FLAGS} ${BINARY_SRC}
	@printf "$(OK_COLOR)==> Building ${BINARY} for ${BUILD_GOOS}/${BUILD_GOARCH} succeed $(NO_COLOR)\n"

test-unit:
	@printf "$(OK_COLOR)==> Running unit tests$(NO_COLOR)\n"
	@CGO_ENABLED=0 GOFLAGS=-mod=vendor go test -cover ./... -coverprofile=coverage_unit.txt -covermode=atomic

# Added -p=1 to fix flakiness in integration DB tests
test-integration: clean build
	@printf "$(OK_COLOR)==> Running integration tests$(NO_COLOR)\n"
	@CGO_ENABLED=0 GOFLAGS=-mod=vendor go test -tags integration -cover ./... -coverprofile=coverage_integration.txt -covermode=atomic

# Cleans our project: deletes binaries
clean:
	@printf "$(OK_COLOR)==> Cleaning project$(NO_COLOR)\n"
	if [ -d ${BUILD_DIR} ] ; then rm -rf ${BUILD_DIR}/* ; fi

SHA=$(shell git rev-parse --short HEAD)
VERSION=$(shell cat VERSION)
DIRTY=false
# TODO add release flag
GO_PACKAGE=$(shell go list)
LDFLAGS=-ldflags "-w -s -X $(GO_PACKAGE)/util.GitSha=${SHA} -X $(GO_PACKAGE)/util.Version=${VERSION} -X $(GO_PACKAGE)/util.Dirty=${DIRTY}"
export GO111MODULE=on

all: test install

fmt:
	go install golang.org/x/tools/cmd/goimports@latest
	goimports -w -l .
.PHONY: fmt

build: fmt ## build the binary
	go build ${LDFLAGS} .
.PHONY: build

coverage: ## run the go coverage tool, reading file coverage.out
	go tool cover -html=coverage.out
.PHONY: coverage

lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint --version
.PHONY: lint

test: fmt ## run tests
 ifeq (, $(shell which gotest))
	go test -failfast -cover ./...
 else
	gotest -failfast -cover ./...
 endif
.PHONY: test

test-coverage: ## run the test with proper coverage reporting
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out
.PHONY: test-coverage

install: ## install the camelot binary in $GOPATH/bin
	go install ${LDFLAGS} .
.PHONY: install

help: ## display help for this makefile
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help

clean: ## clean the repo
	rm camelot 2>/dev/null || true
	go clean
	go clean -testcache
	rm -rf dist 2>/dev/null || true
	rm coverage.out 2>/dev/null || true
.PHONY: clean

generate-mocks:
	rm -rf ./mocks/*
	go install github.com/golang/mock/mockgen/...@v1.6.0
	go generate -x ./pkg/...
.PHONY: generate-mocks
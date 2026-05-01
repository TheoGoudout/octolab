.PHONY: all test test-dispatcher test-octoshim lint lint-shell fmt docker-build clean help

SHIM_IMAGE   ?= octolab/shim-runner:dev
PYTHON       ?= python3
GO           ?= go

all: test

## test: run all tests (dispatcher + octoshim)
test: test-dispatcher test-octoshim

## test-dispatcher: run Python dispatcher unit tests
test-dispatcher:
	$(PYTHON) -m pytest dispatcher/tests/ -v

## test-octoshim: run Go proxy unit tests
test-octoshim:
	cd octoshim && $(GO) test ./... -v -count=1

## lint: run all linters
lint: lint-shell lint-go

## lint-shell: lint shell scripts with shellcheck
lint-shell:
	shellcheck entrypoint/bridge.sh entrypoint/mask-tokens.sh

## lint-go: run go vet on the proxy
lint-go:
	cd octoshim && $(GO) vet ./...

## fmt: format Go source files
fmt:
	cd octoshim && $(GO) fmt ./...

## docker-build: build the shim runner Docker image
docker-build:
	docker build --target runtime --tag $(SHIM_IMAGE) .

## docker-build-no-cache: build the shim runner image without cache
docker-build-no-cache:
	docker build --no-cache --target runtime --tag $(SHIM_IMAGE) .

## octoshim-build: build the octoshim binary locally
octoshim-build:
	cd octoshim && CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o ../bin/octoshim .

## dispatcher-install: install Python dependencies for the dispatcher
dispatcher-install:
	pip install -r dispatcher/requirements.txt

## clean: remove build artifacts
clean:
	rm -rf bin/ generated-ci.yml bridge-logs/ .pytest_cache/
	find . -name '__pycache__' -exec rm -rf {} + 2>/dev/null || true

## help: print this help message
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //'

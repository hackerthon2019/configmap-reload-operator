## vortex server version
OPERATOR_VERSION = v0.1.0
## Folder content generated files
BUILD_FOLDER = ./bin
PROJECT_URL  = github.com/hackerthon2019/configmap-reload-operator
## command
GO           = go
GO_VENDOR    = govendor
MKDIR_P      = mkdir -p

## Random Alphanumeric String
SECRET_KEY   = $(shell cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)

## UNAME
UNAME := $(shell uname)

################################################

.PHONY: all
all: build test


.phony: build
build:
	$(make) src.build

.phony: run
run:
	/usr/local/bin/operator-sdk up local --namespace=default

.PHONY: test
test: build
	$(MAKE) src.test

.PHONY: clean
clean:
	$(RM) -rf $(BUILD_FOLDER)/cmd

## src/ ########################################

.PHONY: src.build
src.build:
	$(MKDIR_P) $(BUILD_FOLDER)/cmd/
	$(GO) build -v -o $(BUILD_FOLDER)/cmd/manager ./cmd/manager/main.go

.PHONY: src.test
src.test:
	$(GO) test -v ./src/...

.PHONY: src.install
src.install:
	$(GO) install -v ./src/...

.PHONY: src.test-coverage
src.test-coverage:
	$(MKDIR_P) $(BUILD_FOLDER)/src/
	$(GO) test -v -coverprofile=$(BUILD_FOLDER)/src/coverage.txt -covermode=atomic ./src/...
	$(GO) tool cover -html=$(BUILD_FOLDER)/src/coverage.txt -o $(BUILD_FOLDER)/src/coverage.html

## launch apps #############################

## dockerfiles/ ########################################

.PHONY: dockerfiles.build
dockerfiles.build:
	docker build --tag hackerthon2019/cm-reload-operator:$(OPERATOR_VERSION) .

## git tag version ########################################

.PHONY: push.tag
push.tag:
	@echo "Current git tag version:"$(OPERATOR_VERSION)
	git tag $(OPERATOR_VERSION)
	git push --tags

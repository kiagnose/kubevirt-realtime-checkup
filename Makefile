CONTAINER_ENGINE ?= podman

CHECKUP_IMAGE_NAME ?= quay.io/kiagnose/kubevirt-rt-checkup
CHECKUP_IMAGE_TAG ?= devel

GO_IMAGE_NAME := docker.io/library/golang
GO_IMAGE_TAG := 1.19.4-bullseye

LINTER_IMAGE_NAME := docker.io/golangci/golangci-lint
LINTER_IMAGE_TAG := v1.50.1

PROJECT_WORKING_DIR := /go/src/github.com/kiagnose/kubevirt-rt-checkup

all: lint unit-test build
.PHONY: all

build:
	$(CONTAINER_ENGINE) run --rm \
		-v $(PWD):$(PROJECT_WORKING_DIR):Z \
		--workdir $(PROJECT_WORKING_DIR) \
		$(GO_IMAGE_NAME):$(GO_IMAGE_TAG) \
		go build -v -o ./bin/kubevirt-rt-checkup ./cmd/

	$(CONTAINER_ENGINE) build . -t $(CHECKUP_IMAGE_NAME):$(CHECKUP_IMAGE_TAG)
.PHONY: build

unit-test:
	$(CONTAINER_ENGINE) run --rm \
		-v $(PWD):$(PROJECT_WORKING_DIR):Z \
		--workdir $(PROJECT_WORKING_DIR) \
		$(GO_IMAGE_NAME):$(GO_IMAGE_TAG) \
		go test -v ./cmd/... ./pkg/...
.PHONY: unit-test

lint:
	mkdir -p $(PWD)/linter-cache

	$(CONTAINER_ENGINE) run --rm \
		-v $(PWD):$(PROJECT_WORKING_DIR):Z \
		-v $(PWD)/linter-cache:/root/.cache:Z \
		--workdir $(PROJECT_WORKING_DIR) \
		$(LINTER_IMAGE_NAME):$(LINTER_IMAGE_TAG) \
		golangci-lint run -v --timeout=3m ./cmd/... ./pkg/...
.PHONY: lint

push:
	$(CONTAINER_ENGINE) push $(CHECKUP_IMAGE_NAME):$(CHECKUP_IMAGE_TAG)
.PHONY: push

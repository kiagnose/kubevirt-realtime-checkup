CONTAINER_ENGINE ?= podman

CHECKUP_IMAGE_NAME := quay.io/kiagnose/kubevirt-rt-checkup
CHECKUP_IMAGE_TAG ?= devel

GO_IMAGE_NAME := docker.io/library/golang
GO_IMAGE_TAG := 1.19.4-alpine3.17

PROJECT_WORKING_DIR := /go/src/github.com/kiagnose/kubevirt-rt-checkup

build:
	$(CONTAINER_ENGINE) run --rm -v $(PWD):$(PROJECT_WORKING_DIR):Z --workdir $(PROJECT_WORKING_DIR) $(GO_IMAGE_NAME):$(GO_IMAGE_TAG) go build -v -o ./bin/kubevirt-rt-checkup ./cmd/
	$(CONTAINER_ENGINE) build . -t $(CHECKUP_IMAGE_NAME):$(CHECKUP_IMAGE_TAG)
.PHONY: build

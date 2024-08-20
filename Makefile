CONTAINER_ENGINE ?= podman

REG ?= quay.io
ORG ?= kiagnose

CHECKUP_IMAGE_NAME ?= $(REG)/$(ORG)/kubevirt-realtime-checkup
CHECKUP_IMAGE_TAG ?= devel

VM_IMAGE_BUILDER_IMAGE_NAME := kubevirt-realtime-checkup-vm-image-builder
VM_IMAGE_BUILDER_IMAGE_TAG ?= latest

VIRT_BUILDER_CACHE_DIR := $(CURDIR)/_virt_builder/cache
VIRT_BUILDER_OUTPUT_DIR := $(CURDIR)/_virt_builder/output

VM_CONTAINER_DISK_IMAGE_NAME := kubevirt-realtime-checkup-vm
VM_CONTAINER_DISK_IMAGE_TAG ?= latest

GO_IMAGE_NAME := docker.io/library/golang
GO_IMAGE_TAG := 1.20.12

LINTER_IMAGE_NAME := docker.io/golangci/golangci-lint
LINTER_IMAGE_TAG := v1.60.1
KUBECONFIG ?= $(HOME)/.kube/config

PROJECT_WORKING_DIR := /go/src/github.com/kiagnose/kubevirt-realtime-checkup

E2E_TEST_TIMEOUT ?= 1h
E2E_TEST_ARGS ?= -test.v -test.timeout=$(E2E_TEST_TIMEOUT) -ginkgo.v -ginkgo.timeout=$(E2E_TEST_TIMEOUT) $(E2E_TEST_EXTRA_ARGS)

all: lint unit-test build
.PHONY: all

build:
	mkdir -p $(PWD)/_go-cache

	$(CONTAINER_ENGINE) run --rm \
		-v $(PWD):$(PROJECT_WORKING_DIR):Z \
		-v $(PWD)/_go-cache:/root/.cache/go-build:Z \
		--workdir $(PROJECT_WORKING_DIR) \
		$(GO_IMAGE_NAME):$(GO_IMAGE_TAG) \
		go build -v -o ./bin/kubevirt-realtime-checkup ./cmd/

	$(CONTAINER_ENGINE) build . -t $(CHECKUP_IMAGE_NAME):$(CHECKUP_IMAGE_TAG)
.PHONY: build

vendor-deps:
	mkdir -p $(PWD)/_go-cache

	$(CONTAINER_ENGINE) run --rm \
		-v $(PWD):$(PROJECT_WORKING_DIR):Z \
		-v $(PWD)/_go-cache:/root/.cache/go-build:Z \
		--workdir $(PROJECT_WORKING_DIR) \
		$(GO_IMAGE_NAME):$(GO_IMAGE_TAG) \
		go mod tidy && go mod vendor
.PHONY: vendor-deps

unit-test:
	mkdir -p $(PWD)/_go-cache

	$(CONTAINER_ENGINE) run --rm \
		-v $(PWD):$(PROJECT_WORKING_DIR):Z \
		-v $(PWD)/_go-cache:/root/.cache/go-build:Z \
		--workdir $(PROJECT_WORKING_DIR) \
		$(GO_IMAGE_NAME):$(GO_IMAGE_TAG) \
		go test -v ./cmd/... ./pkg/...
.PHONY: unit-test

e2e-test:
	mkdir -p $(PWD)/_go-cache

	$(CONTAINER_ENGINE) run --rm \
		-v $(PWD):$(PROJECT_WORKING_DIR):Z \
		-v $(PWD)/_go-cache:/root/.cache/go-build:Z \
		-v $(shell dirname $(KUBECONFIG)):/root/.kube:Z,ro \
		--workdir $(PROJECT_WORKING_DIR) \
		-e KUBECONFIG=/root/.kube/$(shell basename $(KUBECONFIG)) \
		-e TEST_NAMESPACE=$(TEST_NAMESPACE) \
		-e TEST_CHECKUP_IMAGE=$(TEST_CHECKUP_IMAGE) \
		-e VM_UNDER_TEST_CONTAINER_DISK_IMAGE=$(VM_UNDER_TEST_CONTAINER_DISK_IMAGE) \
		$(GO_IMAGE_NAME):$(GO_IMAGE_TAG) \
		go test -v ./tests/... $(E2E_TEST_ARGS)
.PHONY: e2e-test

lint:
	mkdir -p $(PWD)/_linter-cache

	$(CONTAINER_ENGINE) run --rm \
		-v $(PWD):$(PROJECT_WORKING_DIR):Z \
		-v $(PWD)/_linter-cache:/root/.cache:Z \
		--workdir $(PROJECT_WORKING_DIR) \
		$(LINTER_IMAGE_NAME):$(LINTER_IMAGE_TAG) \
		golangci-lint run -v --timeout=3m ./cmd/... ./pkg/... ./tests...
.PHONY: lint

push:
	$(CONTAINER_ENGINE) push $(CHECKUP_IMAGE_NAME):$(CHECKUP_IMAGE_TAG)
.PHONY: push

build-vm-image-builder:
	$(CONTAINER_ENGINE) build $(CURDIR)/vms/image-builder -f $(CURDIR)/vms/image-builder/Dockerfile -t $(REG)/$(ORG)/$(VM_IMAGE_BUILDER_IMAGE_NAME):$(VM_IMAGE_BUILDER_IMAGE_TAG)
.PHONY: build-vm-image-builder

build-vm-image: build-vm-image-builder
	mkdir -vp $(VIRT_BUILDER_CACHE_DIR)
	mkdir -vp $(VIRT_BUILDER_OUTPUT_DIR)

	$(CONTAINER_ENGINE) container run --rm \
      --volume=$(VIRT_BUILDER_CACHE_DIR):/root/.cache/virt-builder:Z \
      --volume=$(VIRT_BUILDER_OUTPUT_DIR):/output:Z \
      --volume=$(CURDIR)/vms/vm-under-test/scripts:/root/scripts:Z \
      $(REG)/$(ORG)/$(VM_IMAGE_BUILDER_IMAGE_NAME):$(VM_IMAGE_BUILDER_IMAGE_TAG) \
      /root/scripts/build-vm-image
.PHONY: build-vm-image

build-vm-container-disk: build-vm-image
	$(CONTAINER_ENGINE) build $(CURDIR) -f $(CURDIR)/vms/vm-under-test/Dockerfile -t $(REG)/$(ORG)/$(VM_CONTAINER_DISK_IMAGE_NAME):$(VM_CONTAINER_DISK_IMAGE_TAG)
.PHONY: build-vm-container-disk

push-vm-container-disk:
	$(CONTAINER_ENGINE) push $(REG)/$(ORG)/$(VM_CONTAINER_DISK_IMAGE_NAME):$(VM_CONTAINER_DISK_IMAGE_TAG)
.PHONY: push-vm-container-disk

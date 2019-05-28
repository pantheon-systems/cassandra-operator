APP := cassandra-operator
VERSION := 0.1.0
UNIQUE_TAG ?= $(shell git rev-parse HEAD)
SDK_RELEASE_URL :=  https://github.com/operator-framework/operator-sdk/releases/download/v0.1.0/operator-sdk-v0.1.0-x86_64-linux-gnu

include _go.mk

NAMESPACE := "kube-system"
ifdef KUBE_NAMESPACE
	NAMESPACE = $(KUBE_NAMESPACE)
endif

ifndef KUBE_CONTEXT
	KUBE_CONTEXT := gke_pantheon-dev_us-central1-b_sandbox-01
endif

ifdef CIRCLE_WORKFLOW_ID
	UNIQUE_TAG = $(CIRCLE_WORKFLOW_ID)
endif

ifdef CIRCLE_WORKFLOW_ID
  BUILD_NUM := $(CIRCLE_WORKFLOW_ID)
  ifeq (email-required, $(shell docker login --help | grep -q Email && echo email-required))
    QUAY := docker login -p "$$QUAY_PASSWD" -u "$$QUAY_USER" -e "unused@unused" quay.io
  else
    QUAY := docker login -p "$$QUAY_PASSWD" -u "$$QUAY_USER" quay.io
  endif
endif

deps::
	@dep ensure -v

build:
	@operator-sdk build quay.io/getpantheon/cassandra-operator:v$(VERSION)-$(UNIQUE_TAG)

push: setup-quay
	@docker push quay.io/getpantheon/cassandra-operator:v$(VERSION)-$(UNIQUE_TAG)

generate:
	@operator-sdk generate k8s

install-sdk:
	@curl -L $(SDK_RELEASE_URL) -o $(GOPATH)/bin/operator-sdk
	@chmod 755 $(GOPATH)/bin/operator-sdk

version:
	@echo $(VERSION)-$(UNIQUE_TAG)

setup-quay:: ## setup docker login for quay.io
ifdef CIRCLE_BUILD_NUM
ifndef QUAY_PASSWD
		$(call ERROR, "Need to set QUAY_PASSWD environment variable.")
endif
ifndef QUAY_USER
		$(call ERROR, "Need to set QUAY_USER environment variable.")
endif
	$(call INFO, "Setting up quay login credentials.")
	@$(QUAY) > /dev/null
endif

.PHONY:: setup-quay push deploy

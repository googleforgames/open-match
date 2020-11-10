# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

## Open Match Make Help
## ====================
##
## # Create a GKE Cluster (requires gcloud installed and initialized, https://cloud.google.com/sdk/docs/quickstarts)
## make activate-gcp-apis
## make create-gke-cluster push-helm
##
## # Create a Minikube Cluster (requires VirtualBox)
## make create-mini-cluster push-helm
##
## # Create a KinD Cluster (Follow instructions to run command before pushing helm.)
## make create-kind-cluster get-kind-kubeconfig
##
## # Finish KinD setup by installing helm:
## make push-helm
##
## # Deploy Open Match
## make push-images -j$(nproc)
## make install-chart
##
## # Build and Test
## make all -j$(nproc)
## make test
##
## # Access telemetry
## make proxy-prometheus
## make proxy-grafana
## make proxy-ui
##
## # Teardown
## make delete-mini-cluster
## make delete-gke-cluster
## make delete-kind-cluster && export KUBECONFIG=""
##
## # Prepare a Pull Request
## make presubmit
##

# If you want information on how to edit this file checkout,
# http://makefiletutorial.com/

BASE_VERSION = 0.0.0-dev
SHORT_SHA = $(shell git rev-parse --short=7 HEAD | tr -d [:punct:])
BRANCH_NAME = $(shell git rev-parse --abbrev-ref HEAD | tr -d [:punct:])
VERSION = $(BASE_VERSION)-$(SHORT_SHA)
BUILD_DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
YEAR_MONTH = $(shell date -u +'%Y%m')
YEAR_MONTH_DAY = $(shell date -u +'%Y%m%d')
MAJOR_MINOR_VERSION = $(shell echo $(BASE_VERSION) | cut -d '.' -f1).$(shell echo $(BASE_VERSION) | cut -d '.' -f2)
PROTOC_VERSION = 3.10.1
HELM_VERSION = 3.0.0
KUBECTL_VERSION = 1.16.2
MINIKUBE_VERSION = latest
GOLANGCI_VERSION = 1.18.0
KIND_VERSION = 0.5.1
SWAGGERUI_VERSION = 3.24.2
GOOGLE_APIS_VERSION = aba342359b6743353195ca53f944fe71e6fb6cd4
GRPC_GATEWAY_VERSION = 1.14.3
TERRAFORM_VERSION = 0.12.13
CHART_TESTING_VERSION = 2.4.0

# A workaround to simplify Open Match development workflow
REDIS_DEV_PASSWORD = helloworld

ENABLE_SECURITY_HARDENING = 0
GO = GO111MODULE=on go
# Defines the absolute local directory of the open-match project
REPOSITORY_ROOT := $(patsubst %/,%,$(dir $(abspath $(MAKEFILE_LIST))))
BUILD_DIR = $(REPOSITORY_ROOT)/build
TOOLCHAIN_DIR = $(BUILD_DIR)/toolchain
TOOLCHAIN_BIN = $(TOOLCHAIN_DIR)/bin
PROTOC_INCLUDES := $(REPOSITORY_ROOT)/third_party
GCP_PROJECT_ID ?=
GCP_PROJECT_FLAG = --project=$(GCP_PROJECT_ID)
OPEN_MATCH_BUILD_PROJECT_ID = open-match-build
OPEN_MATCH_PUBLIC_IMAGES_PROJECT_ID = open-match-public-images
REGISTRY ?= gcr.io/$(GCP_PROJECT_ID)
TAG = $(VERSION)
ALTERNATE_TAG = dev
VERSIONED_CANARY_TAG = $(BASE_VERSION)-canary
DATED_CANARY_TAG = $(YEAR_MONTH_DAY)-canary
CANARY_TAG = canary
GKE_CLUSTER_NAME = om-cluster
GCP_REGION = us-west1
GCP_ZONE = us-west1-a
GCP_LOCATION = $(GCP_ZONE)
EXE_EXTENSION =
GCP_LOCATION_FLAG = --zone $(GCP_ZONE)
GO111MODULE = on
GOLANG_TEST_COUNT = 1
SWAGGERUI_PORT = 51500
PROMETHEUS_PORT = 9090
JAEGER_QUERY_PORT = 16686
GRAFANA_PORT = 3000
FRONTEND_PORT = 51504
BACKEND_PORT = 51505
QUERY_PORT = 51503
SYNCHRONIZER_PORT = 51506
DEMO_PORT = 51507
PROTOC := $(TOOLCHAIN_BIN)/protoc$(EXE_EXTENSION)
HELM = $(TOOLCHAIN_BIN)/helm$(EXE_EXTENSION)
MINIKUBE = $(TOOLCHAIN_BIN)/minikube$(EXE_EXTENSION)
KUBECTL = $(TOOLCHAIN_BIN)/kubectl$(EXE_EXTENSION)
KIND = $(TOOLCHAIN_BIN)/kind$(EXE_EXTENSION)
TERRAFORM = $(TOOLCHAIN_BIN)/terraform$(EXE_EXTENSION)
CERTGEN = $(TOOLCHAIN_BIN)/certgen$(EXE_EXTENSION)
GOLANGCI = $(TOOLCHAIN_BIN)/golangci-lint$(EXE_EXTENSION)
CHART_TESTING = $(TOOLCHAIN_BIN)/ct$(EXE_EXTENSION)
GCLOUD = gcloud --quiet
OPEN_MATCH_HELM_NAME = open-match
OPEN_MATCH_KUBERNETES_NAMESPACE = open-match
OPEN_MATCH_SECRETS_DIR = $(REPOSITORY_ROOT)/install/helm/open-match/secrets
GCLOUD_ACCOUNT_EMAIL = $(shell gcloud auth list --format yaml | grep ACTIVE -a2 | grep account: | cut -c 10-)
_GCB_POST_SUBMIT ?= 0
# Latest version triggers builds of :latest images.
_GCB_LATEST_VERSION ?= undefined
IMAGE_BUILD_ARGS = --build-arg BUILD_DATE=$(BUILD_DATE) --build-arg=VCS_REF=$(SHORT_SHA) --build-arg BUILD_VERSION=$(BASE_VERSION)
GCLOUD_EXTRA_FLAGS =
# Make port forwards accessible outside of the proxy machine.
PORT_FORWARD_ADDRESS_FLAG = --address 0.0.0.0
DASHBOARD_PORT = 9092

# Open Match Cluster E2E Test Variables
OPEN_MATCH_CI_LABEL = open-match-ci

# This flag is set when running in Continuous Integration.
ifdef OPEN_MATCH_CI_MODE
	export KUBECONFIG = $(HOME)/.kube/config
	GCLOUD = gcloud --quiet --no-user-output-enabled
	GKE_CLUSTER_NAME = open-match-ci
endif

export PATH := $(TOOLCHAIN_BIN):$(PATH)

# Get the project from gcloud if it's not set.
ifeq ($(GCP_PROJECT_ID),)
	export GCP_PROJECT_ID = $(shell gcloud config list --format 'value(core.project)')
endif

ifeq ($(OS),Windows_NT)
	HELM_PACKAGE = https://get.helm.sh/helm-v$(HELM_VERSION)-windows-amd64.zip
	MINIKUBE_PACKAGE = https://storage.googleapis.com/minikube/releases/$(MINIKUBE_VERSION)/minikube-windows-amd64.exe
	EXE_EXTENSION = .exe
	PROTOC_PACKAGE = https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-win64.zip
	KUBECTL_PACKAGE = https://storage.googleapis.com/kubernetes-release/release/v$(KUBECTL_VERSION)/bin/windows/amd64/kubectl.exe
	GOLANGCI_PACKAGE = https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_VERSION)/golangci-lint-$(GOLANGCI_VERSION)-windows-amd64.zip
	KIND_PACKAGE = https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-windows-amd64
	TERRAFORM_PACKAGE = https://releases.hashicorp.com/terraform/$(TERRAFORM_VERSION)/terraform_$(TERRAFORM_VERSION)_windows_amd64.zip
	CHART_TESTING_PACKAGE = https://github.com/helm/chart-testing/releases/download/v$(CHART_TESTING_VERSION)/chart-testing_$(CHART_TESTING_VERSION)_windows_amd64.zip
	SED_REPLACE = sed -i
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		HELM_PACKAGE = https://get.helm.sh/helm-v$(HELM_VERSION)-linux-amd64.tar.gz
		MINIKUBE_PACKAGE = https://storage.googleapis.com/minikube/releases/$(MINIKUBE_VERSION)/minikube-linux-amd64
		PROTOC_PACKAGE = https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-linux-x86_64.zip
		KUBECTL_PACKAGE = https://storage.googleapis.com/kubernetes-release/release/v$(KUBECTL_VERSION)/bin/linux/amd64/kubectl
		GOLANGCI_PACKAGE = https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_VERSION)/golangci-lint-$(GOLANGCI_VERSION)-linux-amd64.tar.gz
		KIND_PACKAGE = https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-linux-amd64
		TERRAFORM_PACKAGE = https://releases.hashicorp.com/terraform/$(TERRAFORM_VERSION)/terraform_$(TERRAFORM_VERSION)_linux_amd64.zip
		CHART_TESTING_PACKAGE = https://github.com/helm/chart-testing/releases/download/v$(CHART_TESTING_VERSION)/chart-testing_$(CHART_TESTING_VERSION)_linux_amd64.tar.gz
		SED_REPLACE = sed -i
	endif
	ifeq ($(UNAME_S),Darwin)
		HELM_PACKAGE = https://get.helm.sh/helm-v$(HELM_VERSION)-darwin-amd64.tar.gz
		MINIKUBE_PACKAGE = https://storage.googleapis.com/minikube/releases/$(MINIKUBE_VERSION)/minikube-darwin-amd64
		PROTOC_PACKAGE = https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-osx-x86_64.zip
		KUBECTL_PACKAGE = https://storage.googleapis.com/kubernetes-release/release/v$(KUBECTL_VERSION)/bin/darwin/amd64/kubectl
		GOLANGCI_PACKAGE = https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_VERSION)/golangci-lint-$(GOLANGCI_VERSION)-darwin-amd64.tar.gz
		KIND_PACKAGE = https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-darwin-amd64
		TERRAFORM_PACKAGE = https://releases.hashicorp.com/terraform/$(TERRAFORM_VERSION)/terraform_$(TERRAFORM_VERSION)_darwin_amd64.zip
		CHART_TESTING_PACKAGE = https://github.com/helm/chart-testing/releases/download/v$(CHART_TESTING_VERSION)/chart-testing_$(CHART_TESTING_VERSION)_darwin_amd64.tar.gz
		SED_REPLACE = sed -i ''
	endif
endif

GOLANG_PROTOS = pkg/pb/backend.pb.go pkg/pb/frontend.pb.go pkg/pb/matchfunction.pb.go pkg/pb/query.pb.go pkg/pb/messages.pb.go pkg/pb/extensions.pb.go pkg/pb/evaluator.pb.go internal/ipb/synchronizer.pb.go internal/ipb/internal.pb.go pkg/pb/backend.pb.gw.go pkg/pb/frontend.pb.gw.go pkg/pb/matchfunction.pb.gw.go pkg/pb/query.pb.gw.go pkg/pb/evaluator.pb.gw.go

SWAGGER_JSON_DOCS = api/frontend.swagger.json api/backend.swagger.json api/query.swagger.json api/matchfunction.swagger.json api/evaluator.swagger.json

ALL_PROTOS = $(GOLANG_PROTOS) $(SWAGGER_JSON_DOCS)

# CMDS is a list of all folders in cmd/
CMDS = $(notdir $(wildcard cmd/*))

# Names of the individual images, ommiting the openmatch prefix.
IMAGES = $(CMDS) mmf-go-soloduel base-build

help:
	@cat Makefile | grep ^\#\# | grep -v ^\#\#\# |cut -c 4-

local-cloud-build: LOCAL_CLOUD_BUILD_PUSH = # --push
local-cloud-build: gcloud
	cloud-build-local --config=cloudbuild.yaml --dryrun=false $(LOCAL_CLOUD_BUILD_PUSH) --substitutions SHORT_SHA=$(SHORT_SHA),_GCB_POST_SUBMIT=$(_GCB_POST_SUBMIT),_GCB_LATEST_VERSION=$(_GCB_LATEST_VERSION),BRANCH_NAME=$(BRANCH_NAME) .

################################################################################
## #############################################################################
## Image commands:
## These commands are auto-generated based on a complete list of images.
## All folders in cmd/ are turned into an image using Dockerfile.cmd.
## Additional images are specified by the IMAGES variable.
## Image commands ommit the "openmatch-" prefix on the image name and tags.
##

list-images:
	@echo $(IMAGES)

#######################################
## # Builds images locally
## build-images / build-<image name>-image
##
build-images: $(foreach IMAGE,$(IMAGES),build-$(IMAGE)-image)

# Include all-protos here so that all dependencies are guaranteed to be downloaded after the base image is created.
# This is important so that the repository does not have any mutations while building individual images.
build-base-build-image: docker $(ALL_PROTOS)
	docker build -f Dockerfile.base-build -t open-match-base-build -t $(REGISTRY)/openmatch-base-build:$(TAG) -t $(REGISTRY)/openmatch-base-build:$(ALTERNATE_TAG) .

$(foreach CMD,$(CMDS),build-$(CMD)-image): build-%-image: docker build-base-build-image
	docker build \
		-f Dockerfile.cmd \
		$(IMAGE_BUILD_ARGS) \
		--build-arg=IMAGE_TITLE=$* \
		-t $(REGISTRY)/openmatch-$*:$(TAG) \
		-t $(REGISTRY)/openmatch-$*:$(ALTERNATE_TAG) \
		.

build-mmf-go-soloduel-image: docker build-base-build-image
	docker build -f examples/functions/golang/soloduel/Dockerfile -t $(REGISTRY)/openmatch-mmf-go-soloduel:$(TAG) -t $(REGISTRY)/openmatch-mmf-go-soloduel:$(ALTERNATE_TAG) .

#######################################
## # Builds and pushes images to your container registry.
## push-images / push-<image name>-image
##
push-images: $(foreach IMAGE,$(IMAGES),push-$(IMAGE)-image)

$(foreach IMAGE,$(IMAGES),push-$(IMAGE)-image): push-%-image: build-%-image docker
	docker push $(REGISTRY)/openmatch-$*:$(TAG)
	docker push $(REGISTRY)/openmatch-$*:$(ALTERNATE_TAG)
ifeq ($(_GCB_POST_SUBMIT),1)
	docker tag $(REGISTRY)/openmatch-$*:$(TAG) $(REGISTRY)/openmatch-$*:$(VERSIONED_CANARY_TAG)
	docker push $(REGISTRY)/openmatch-$*:$(VERSIONED_CANARY_TAG)
ifeq ($(BASE_VERSION),0.0.0-dev)
	docker tag $(REGISTRY)/openmatch-$*:$(TAG) $(REGISTRY)/openmatch-$*:$(DATED_CANARY_TAG)
	docker push $(REGISTRY)/openmatch-$*:$(DATED_CANARY_TAG)
	docker tag $(REGISTRY)/openmatch-$*:$(TAG) $(REGISTRY)/openmatch-$*:$(CANARY_TAG)
	docker push $(REGISTRY)/openmatch-$*:$(CANARY_TAG)
endif
endif

#######################################
## # Publishes images on the public container registry.
## # Used for publishing releases.
## retag-images / retag-<image name>-image
##
retag-images: $(foreach IMAGE,$(IMAGES),retag-$(IMAGE)-image)

retag-%-image: SOURCE_REGISTRY = gcr.io/$(OPEN_MATCH_BUILD_PROJECT_ID)
retag-%-image: TARGET_REGISTRY = gcr.io/$(OPEN_MATCH_PUBLIC_IMAGES_PROJECT_ID)
retag-%-image: SOURCE_TAG = canary
$(foreach IMAGE,$(IMAGES),retag-$(IMAGE)-image): retag-%-image: docker
	docker pull $(SOURCE_REGISTRY)/openmatch-$*:$(SOURCE_TAG)
	docker tag $(SOURCE_REGISTRY)/openmatch-$*:$(SOURCE_TAG) $(TARGET_REGISTRY)/openmatch-$*:$(TAG)
	docker push $(TARGET_REGISTRY)/openmatch-$*:$(TAG)

#######################################
## # Removes images from local docker
## clean-images / clean-<image name>-image
##
clean-images: docker $(foreach IMAGE,$(IMAGES),clean-$(IMAGE)-image)
	-docker rmi -f open-match-base-build

$(foreach IMAGE,$(IMAGES),clean-$(IMAGE)-image): clean-%-image:
	-docker rmi -f $(REGISTRY)/openmatch-$*:$(TAG) $(REGISTRY)/openmatch-$*:$(ALTERNATE_TAG)

#####################################################################################################################
update-chart-deps: build/toolchain/bin/helm$(EXE_EXTENSION)
	(cd $(REPOSITORY_ROOT)/install/helm/open-match; $(HELM) repo add incubator https://kubernetes-charts-incubator.storage.googleapis.com; $(HELM) dependency update)

lint-chart: build/toolchain/bin/helm$(EXE_EXTENSION) build/toolchain/bin/ct$(EXE_EXTENSION)
	(cd $(REPOSITORY_ROOT)/install/helm; $(HELM) lint $(OPEN_MATCH_HELM_NAME))
	$(CHART_TESTING) lint --all --chart-yaml-schema $(TOOLCHAIN_BIN)/etc/chart_schema.yaml --lint-conf $(TOOLCHAIN_BIN)/etc/lintconf.yaml --chart-dirs $(REPOSITORY_ROOT)/install/helm/
	$(CHART_TESTING) lint-and-install --all --chart-yaml-schema $(TOOLCHAIN_BIN)/etc/chart_schema.yaml --lint-conf $(TOOLCHAIN_BIN)/etc/lintconf.yaml --chart-dirs $(REPOSITORY_ROOT)/install/helm/

build/chart/open-match-$(BASE_VERSION).tgz: build/toolchain/bin/helm$(EXE_EXTENSION) lint-chart
	mkdir -p $(BUILD_DIR)/chart/
	$(HELM) package -d $(BUILD_DIR)/chart/ --version $(BASE_VERSION) $(REPOSITORY_ROOT)/install/helm/open-match

build/chart/index.yaml: build/toolchain/bin/helm$(EXE_EXTENSION) gcloud build/chart/open-match-$(BASE_VERSION).tgz
	mkdir -p $(BUILD_DIR)/chart-index/
	-gsutil cp gs://open-match-chart/chart/index.yaml $(BUILD_DIR)/chart-index/
	-gsutil -m cp gs://open-match-chart/chart/open-match-* $(BUILD_DIR)/chart-index/
	$(HELM) repo index $(BUILD_DIR)/chart-index/
	$(HELM) repo index --merge $(BUILD_DIR)/chart-index/index.yaml $(BUILD_DIR)/chart/

build/chart/index.yaml.$(YEAR_MONTH_DAY): build/chart/index.yaml
	cp $(BUILD_DIR)/chart/index.yaml $(BUILD_DIR)/chart/index.yaml.$(YEAR_MONTH_DAY)

build/chart/: build/chart/index.yaml build/chart/index.yaml.$(YEAR_MONTH_DAY)

install-chart-prerequisite: build/toolchain/bin/kubectl$(EXE_EXTENSION) update-chart-deps
	-$(KUBECTL) create namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE)
	$(KUBECTL) apply -f install/gke-metadata-server-workaround.yaml

# Used for Open Match development. Install om-configmap-override.yaml by default.
HELM_UPGRADE_FLAGS = --cleanup-on-fail -i --no-hooks --debug --timeout=600s --namespace=$(OPEN_MATCH_KUBERNETES_NAMESPACE) --set global.gcpProjectId=$(GCP_PROJECT_ID) --set open-match-override.enabled=true --set redis.password=$(REDIS_DEV_PASSWORD)
# Used for generate static yamls. Install om-configmap-override.yaml as needed.
HELM_TEMPLATE_FLAGS = --no-hooks --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --set usingHelmTemplate=true
HELM_IMAGE_FLAGS = --set global.image.registry=$(REGISTRY) --set global.image.tag=$(TAG)

install-demo: build/toolchain/bin/helm$(EXE_EXTENSION)
	cp $(REPOSITORY_ROOT)/install/02-open-match-demo.yaml $(REPOSITORY_ROOT)/install/tmp-demo.yaml
	$(SED_REPLACE) 's|gcr.io/open-match-public-images|$(REGISTRY)|g' $(REPOSITORY_ROOT)/install/tmp-demo.yaml
	$(SED_REPLACE) 's|0.0.0-dev|$(TAG)|g' $(REPOSITORY_ROOT)/install/tmp-demo.yaml
	$(KUBECTL) apply -f $(REPOSITORY_ROOT)/install/tmp-demo.yaml
	rm $(REPOSITORY_ROOT)/install/tmp-demo.yaml

# install-large-chart will install open-match-core, open-match-demo with the demo evaluator and mmf, and telemetry supports.
install-large-chart: install-chart-prerequisite install-demo build/toolchain/bin/helm$(EXE_EXTENSION) install/helm/open-match/secrets/
	$(HELM) upgrade $(OPEN_MATCH_HELM_NAME) $(HELM_UPGRADE_FLAGS) --atomic install/helm/open-match $(HELM_IMAGE_FLAGS) \
		--set open-match-telemetry.enabled=true \
		--set open-match-customize.enabled=true \
		--set open-match-customize.evaluator.enabled=true \
		--set global.telemetry.grafana.enabled=true \
		--set global.telemetry.jaeger.enabled=true \
		--set global.telemetry.prometheus.enabled=true

# install-chart will install open-match-core, open-match-demo, with the demo evaluator and mmf.
install-chart: install-chart-prerequisite install-demo build/toolchain/bin/helm$(EXE_EXTENSION) install/helm/open-match/secrets/
	$(HELM) upgrade $(OPEN_MATCH_HELM_NAME) $(HELM_UPGRADE_FLAGS) --atomic install/helm/open-match $(HELM_IMAGE_FLAGS) \
		--set open-match-customize.enabled=true \
		--set open-match-customize.evaluator.enabled=true

# install-scale-chart will wait for installing open-match-core with telemetry supports then install open-match-scale chart.
install-scale-chart: install-chart-prerequisite build/toolchain/bin/helm$(EXE_EXTENSION) install/helm/open-match/secrets/
	$(HELM) upgrade $(OPEN_MATCH_HELM_NAME) $(HELM_UPGRADE_FLAGS) --atomic install/helm/open-match $(HELM_IMAGE_FLAGS) -f install/helm/open-match/values-production.yaml \
		--set open-match-telemetry.enabled=true \
		--set open-match-customize.enabled=true \
		--set open-match-customize.function.enabled=true \
		--set open-match-customize.evaluator.enabled=true \
		--set open-match-customize.function.image=openmatch-scale-mmf \
		--set global.telemetry.grafana.enabled=true \
		--set global.telemetry.jaeger.enabled=false \
		--set global.telemetry.prometheus.enabled=true
	$(HELM) template $(OPEN_MATCH_HELM_NAME)-scale  install/helm/open-match $(HELM_TEMPLATE_FLAGS) $(HELM_IMAGE_FLAGS) -f install/helm/open-match/values-production.yaml \
		--set open-match-core.enabled=false \
		--set open-match-core.redis.enabled=false \
		--set global.telemetry.prometheus.enabled=true \
		--set global.telemetry.grafana.enabled=true \
		--set open-match-scale.enabled=true | $(KUBECTL) apply -f -

# install-ci-chart will install open-match-core with pool based mmf for end-to-end in-cluster test.
install-ci-chart: install-chart-prerequisite build/toolchain/bin/helm$(EXE_EXTENSION) install/helm/open-match/secrets/
	$(HELM) upgrade $(OPEN_MATCH_HELM_NAME) $(HELM_UPGRADE_FLAGS) --atomic install/helm/open-match $(HELM_IMAGE_FLAGS) \
		--set query.replicas=1,frontend.replicas=1,backend.replicas=1 \
		--set evaluator.hostName=open-match-test \
		--set evaluator.grpcPort=50509 \
		--set evaluator.httpPort=51509 \
		--set open-match-core.registrationInterval=200ms \
		--set open-match-core.proposalCollectionInterval=200ms \
		--set open-match-core.assignedDeleteTimeout=200ms \
		--set open-match-core.pendingReleaseTimeout=200ms \
		--set open-match-core.queryPageSize=10 \
		--set global.gcpProjectId=intentionally-invalid-value \
		--set redis.master.resources.requests.cpu=0.6,redis.master.resources.requests.memory=300Mi \
		--set ci=true

delete-chart: build/toolchain/bin/helm$(EXE_EXTENSION) build/toolchain/bin/kubectl$(EXE_EXTENSION)
	-$(HELM) uninstall $(OPEN_MATCH_HELM_NAME)
	-$(HELM) uninstall $(OPEN_MATCH_HELM_NAME)-demo
	-$(KUBECTL) delete psp,clusterrole,clusterrolebinding --selector=release=open-match
	-$(KUBECTL) delete psp,clusterrole,clusterrolebinding --selector=release=open-match-demo
	-$(KUBECTL) delete namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE)
	-$(KUBECTL) delete namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE)-demo

ifneq ($(BASE_VERSION), 0.0.0-dev)
install/yaml/: REGISTRY = gcr.io/$(OPEN_MATCH_PUBLIC_IMAGES_PROJECT_ID)
install/yaml/: TAG = $(BASE_VERSION)
endif
install/yaml/: update-chart-deps install/yaml/install.yaml install/yaml/01-open-match-core.yaml install/yaml/02-open-match-demo.yaml install/yaml/03-prometheus-chart.yaml install/yaml/04-grafana-chart.yaml install/yaml/05-jaeger-chart.yaml install/yaml/06-open-match-override-configmap.yaml install/yaml/07-open-match-default-evaluator.yaml

# We have to hard-code the Jaeger endpoints as we are excluding Jaeger, so Helm cannot determine the endpoints from the Jaeger subchart
install/yaml/01-open-match-core.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template $(OPEN_MATCH_HELM_NAME) $(HELM_TEMPLATE_FLAGS) $(HELM_IMAGE_FLAGS) \
		--set-string global.telemetry.jaeger.agentEndpoint="$(OPEN_MATCH_HELM_NAME)-jaeger-agent:6831" \
		--set-string global.telemetry.jaeger.collectorEndpoint="http://$(OPEN_MATCH_HELM_NAME)-jaeger-collector:14268/api/traces" \
		install/helm/open-match > install/yaml/01-open-match-core.yaml

install/yaml/02-open-match-demo.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	cp $(REPOSITORY_ROOT)/install/02-open-match-demo.yaml $(REPOSITORY_ROOT)/install/yaml/02-open-match-demo.yaml
	$(SED_REPLACE) 's|0.0.0-dev|$(TAG)|g' $(REPOSITORY_ROOT)/install/yaml/02-open-match-demo.yaml
	$(SED_REPLACE) 's|gcr.io/open-match-public-images|$(REGISTRY)|g' $(REPOSITORY_ROOT)/install/yaml/02-open-match-demo.yaml

install/yaml/03-prometheus-chart.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template $(OPEN_MATCH_HELM_NAME) $(HELM_TEMPLATE_FLAGS) $(HELM_IMAGE_FLAGS) \
		--set open-match-core.enabled=false \
		--set open-match-core.redis.enabled=false \
		--set open-match-telemetry.enabled=true \
		--set global.telemetry.prometheus.enabled=true \
		install/helm/open-match > install/yaml/03-prometheus-chart.yaml

# We have to hard-code the Prometheus Server URL as we are excluding Prometheus, so Helm cannot determine the URL from the Prometheus subchart
install/yaml/04-grafana-chart.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template $(OPEN_MATCH_HELM_NAME) $(HELM_TEMPLATE_FLAGS) $(HELM_IMAGE_FLAGS) \
		--set open-match-core.enabled=false \
		--set open-match-core.redis.enabled=false \
		--set open-match-telemetry.enabled=true \
		--set global.telemetry.grafana.enabled=true \
		--set-string global.telemetry.grafana.prometheusServer="http://$(OPEN_MATCH_HELM_NAME)-prometheus-server.$(OPEN_MATCH_KUBERNETES_NAMESPACE).svc.cluster.local:80/" \
		install/helm/open-match > install/yaml/04-grafana-chart.yaml

install/yaml/05-jaeger-chart.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template $(OPEN_MATCH_HELM_NAME) $(HELM_TEMPLATE_FLAGS) $(HELM_IMAGE_FLAGS) \
		--set open-match-core.enabled=false \
		--set open-match-core.redis.enabled=false \
		--set open-match-telemetry.enabled=true \
		--set global.telemetry.jaeger.enabled=true \
		install/helm/open-match > install/yaml/05-jaeger-chart.yaml

install/yaml/06-open-match-override-configmap.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template $(OPEN_MATCH_HELM_NAME) $(HELM_TEMPLATE_FLAGS) $(HELM_IMAGE_FLAGS) \
		--set open-match-core.enabled=false \
		--set open-match-core.redis.enabled=false \
		--set open-match-override.enabled=true \
		-s templates/om-configmap-override.yaml \
		install/helm/open-match > install/yaml/06-open-match-override-configmap.yaml

install/yaml/07-open-match-default-evaluator.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template $(OPEN_MATCH_HELM_NAME) $(HELM_TEMPLATE_FLAGS) $(HELM_IMAGE_FLAGS) \
		--set open-match-core.enabled=false \
		--set open-match-core.redis.enabled=false \
		--set open-match-customize.enabled=true \
		--set open-match-customize.evaluator.enabled=true \
		install/helm/open-match > install/yaml/07-open-match-default-evaluator.yaml

install/yaml/install.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template $(OPEN_MATCH_HELM_NAME) $(HELM_TEMPLATE_FLAGS) $(HELM_IMAGE_FLAGS) \
		--set open-match-customize.enabled=true \
		--set open-match-customize.evaluator.enabled=true \
		--set open-match-telemetry.enabled=true \
		--set global.telemetry.jaeger.enabled=true \
		--set global.telemetry.grafana.enabled=true \
		--set global.telemetry.prometheus.enabled=true \
		install/helm/open-match > install/yaml/install.yaml

set-redis-password:
	@stty -echo; \
		printf "Redis password: "; \
		read REDIS_PASSWORD; \
		stty echo; \
		printf "\n"; \
		$(KUBECTL) create secret generic open-match-redis -n $(OPEN_MATCH_KUBERNETES_NAMESPACE) --from-literal=redis-password=$$REDIS_PASSWORD --dry-run -o yaml | $(KUBECTL) replace -f - --force
## ####################################
## # Tool installation helpers
##

## # Install toolchain. Short for installing K8s, protoc and OpenMatch tools.
## make install-toolchain
##
install-toolchain: install-kubernetes-tools install-protoc-tools install-openmatch-tools

## # Install Kubernetes tools
## make install-kubernetes-tools
##
install-kubernetes-tools: build/toolchain/bin/kubectl$(EXE_EXTENSION) build/toolchain/bin/helm$(EXE_EXTENSION) build/toolchain/bin/minikube$(EXE_EXTENSION) build/toolchain/bin/terraform$(EXE_EXTENSION)

## # Install protoc tools
## make install-protoc-tools
##
install-protoc-tools: build/toolchain/bin/protoc$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-go$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-grpc-gateway$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-swagger$(EXE_EXTENSION)

## # Install OpenMatch tools
## make install-openmatch-tools
##
install-openmatch-tools: build/toolchain/bin/certgen$(EXE_EXTENSION) build/toolchain/bin/reaper$(EXE_EXTENSION)

build/toolchain/bin/helm$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	mkdir -p $(TOOLCHAIN_DIR)/temp-helm
ifeq ($(suffix $(HELM_PACKAGE)),.zip)
	cd $(TOOLCHAIN_DIR)/temp-helm && curl -Lo helm.zip $(HELM_PACKAGE) && unzip -d $(TOOLCHAIN_BIN) -j -q -o helm.zip
else
	cd $(TOOLCHAIN_DIR)/temp-helm && curl -Lo helm.tar.gz $(HELM_PACKAGE) && tar xzf helm.tar.gz -C $(TOOLCHAIN_BIN) --strip-components 1
endif
	rm -rf $(TOOLCHAIN_DIR)/temp-helm/

build/toolchain/bin/ct$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	mkdir -p $(TOOLCHAIN_DIR)/temp-charttesting
ifeq ($(suffix $(CHART_TESTING_PACKAGE)),.zip)
	cd $(TOOLCHAIN_DIR)/temp-charttesting && curl -Lo charttesting.zip $(CHART_TESTING_PACKAGE) && unzip -j -q -o charttesting.zip
else
	cd $(TOOLCHAIN_DIR)/temp-charttesting && curl -Lo charttesting.tar.gz $(CHART_TESTING_PACKAGE) && tar xzf charttesting.tar.gz
endif
	mv $(TOOLCHAIN_DIR)/temp-charttesting/* $(TOOLCHAIN_BIN)
	rm -rf $(TOOLCHAIN_DIR)/temp-charttesting/

build/toolchain/bin/minikube$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	curl -Lo $(MINIKUBE) $(MINIKUBE_PACKAGE)
	chmod +x $(MINIKUBE)

build/toolchain/bin/kubectl$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	curl -Lo $(KUBECTL) $(KUBECTL_PACKAGE)
	chmod +x $(KUBECTL)

build/toolchain/bin/golangci-lint$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	mkdir -p $(TOOLCHAIN_DIR)/temp-golangci
ifeq ($(suffix $(GOLANGCI_PACKAGE)),.zip)
	cd $(TOOLCHAIN_DIR)/temp-golangci && curl -Lo golangci.zip $(GOLANGCI_PACKAGE) && unzip -j -q -o golangci.zip
else
	cd $(TOOLCHAIN_DIR)/temp-golangci && curl -Lo golangci.tar.gz $(GOLANGCI_PACKAGE) && tar xzf golangci.tar.gz --strip-components 1
endif
	mv $(TOOLCHAIN_DIR)/temp-golangci/golangci-lint$(EXE_EXTENSION) $(GOLANGCI)
	rm -rf $(TOOLCHAIN_DIR)/temp-golangci/

build/toolchain/bin/kind$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	curl -Lo $(KIND) $(KIND_PACKAGE)
	chmod +x $(KIND)

build/toolchain/bin/terraform$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	mkdir -p $(TOOLCHAIN_DIR)/temp-terraform
	cd $(TOOLCHAIN_DIR)/temp-terraform && curl -Lo terraform.zip $(TERRAFORM_PACKAGE) && unzip -j -q -o terraform.zip
	mv $(TOOLCHAIN_DIR)/temp-terraform/terraform$(EXE_EXTENSION) $(TOOLCHAIN_BIN)/terraform$(EXE_EXTENSION)
	rm -rf $(TOOLCHAIN_DIR)/temp-terraform/

build/toolchain/bin/protoc$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	curl -o $(TOOLCHAIN_DIR)/protoc-temp.zip -L $(PROTOC_PACKAGE)
	(cd $(TOOLCHAIN_DIR); unzip -q -o protoc-temp.zip)
	rm $(TOOLCHAIN_DIR)/protoc-temp.zip $(TOOLCHAIN_DIR)/readme.txt

build/toolchain/bin/protoc-gen-doc$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	cd $(TOOLCHAIN_BIN) && $(GO) build -i -pkgdir . github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc

build/toolchain/bin/protoc-gen-go$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	cd $(TOOLCHAIN_BIN) && $(GO) build -i -pkgdir . github.com/golang/protobuf/protoc-gen-go

build/toolchain/bin/protoc-gen-grpc-gateway$(EXE_EXTENSION):
	cd $(TOOLCHAIN_BIN) && $(GO) build -i -pkgdir . github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway

build/toolchain/bin/protoc-gen-swagger$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	cd $(TOOLCHAIN_BIN) && $(GO) build -i -pkgdir . github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger

build/toolchain/bin/certgen$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	cd $(TOOLCHAIN_BIN) && $(GO) build $(REPOSITORY_ROOT)/tools/certgen/

build/toolchain/bin/reaper$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	cd $(TOOLCHAIN_BIN) && $(GO) build $(REPOSITORY_ROOT)/tools/reaper/

# Fake target for docker
docker: no-sudo

# Fake target for gcloud
gcloud: no-sudo

tls-certs: install/helm/open-match/secrets/

install/helm/open-match/secrets/: install/helm/open-match/secrets/tls/root-ca/ install/helm/open-match/secrets/tls/server/

install/helm/open-match/secrets/tls/root-ca/: build/toolchain/bin/certgen$(EXE_EXTENSION)
	mkdir -p $(OPEN_MATCH_SECRETS_DIR)/tls/root-ca
	$(CERTGEN) -ca=true -publiccertificate=$(OPEN_MATCH_SECRETS_DIR)/tls/root-ca/public.cert -privatekey=$(OPEN_MATCH_SECRETS_DIR)/tls/root-ca/private.key

install/helm/open-match/secrets/tls/server/: build/toolchain/bin/certgen$(EXE_EXTENSION) install/helm/open-match/secrets/tls/root-ca/
	mkdir -p $(OPEN_MATCH_SECRETS_DIR)/tls/server/
	$(CERTGEN) -publiccertificate=$(OPEN_MATCH_SECRETS_DIR)/tls/server/public.cert -privatekey=$(OPEN_MATCH_SECRETS_DIR)/tls/server/private.key -rootpubliccertificate=$(OPEN_MATCH_SECRETS_DIR)/tls/root-ca/public.cert -rootprivatekey=$(OPEN_MATCH_SECRETS_DIR)/tls/root-ca/private.key

auth-docker: gcloud docker
	$(GCLOUD) $(GCP_PROJECT_FLAG) auth configure-docker

auth-gke-cluster: gcloud
	$(GCLOUD) $(GCP_PROJECT_FLAG) container clusters get-credentials $(GKE_CLUSTER_NAME) $(GCP_LOCATION_FLAG)

activate-gcp-apis: gcloud
	$(GCLOUD) services enable containerregistry.googleapis.com
	$(GCLOUD) services enable container.googleapis.com
	$(GCLOUD) services enable containeranalysis.googleapis.com
	$(GCLOUD) services enable binaryauthorization.googleapis.com

create-gcp-service-account: gcloud
	gcloud $(GCP_PROJECT_FLAG) iam service-accounts create open-match --display-name="Open Match Service Account"
	gcloud $(GCP_PROJECT_FLAG) iam service-accounts add-iam-policy-binding --member=open-match@$(GCP_PROJECT_ID).iam.gserviceaccount.com --role=roles/container.clusterAdmin
	gcloud $(GCP_PROJECT_FLAG) iam service-accounts keys create ~/key.json --iam-account open-match@$(GCP_PROJECT_ID).iam.gserviceaccount.com

create-kind-cluster: build/toolchain/bin/kind$(EXE_EXTENSION) build/toolchain/bin/kubectl$(EXE_EXTENSION)
	$(KIND) create cluster

get-kind-kubeconfig: build/toolchain/bin/kind$(EXE_EXTENSION)
	@echo "============================================="
	@echo "= Run this command"
	@echo "============================================="
	@echo "export KUBECONFIG=\"$(shell $(KIND) get kubeconfig-path)\""
	@echo "============================================="

delete-kind-cluster: build/toolchain/bin/kind$(EXE_EXTENSION) build/toolchain/bin/kubectl$(EXE_EXTENSION)
	-$(KIND) delete cluster

create-cluster-role-binding:
	$(KUBECTL) create clusterrolebinding myname-cluster-admin-binding --clusterrole=cluster-admin --user=$(GCLOUD_ACCOUNT_EMAIL)

create-gke-cluster: GKE_VERSION = 1.14.10-gke.45 # gcloud beta container get-server-config --zone us-west1-a
create-gke-cluster: GKE_CLUSTER_SHAPE_FLAGS = --machine-type n1-standard-4 --enable-autoscaling --min-nodes 1 --num-nodes 2 --max-nodes 10 --disk-size 50
create-gke-cluster: GKE_FUTURE_COMPAT_FLAGS = --no-enable-basic-auth --no-issue-client-certificate --enable-ip-alias --metadata disable-legacy-endpoints=true --enable-autoupgrade
create-gke-cluster: build/toolchain/bin/kubectl$(EXE_EXTENSION) gcloud
	$(GCLOUD) beta $(GCP_PROJECT_FLAG) container clusters create $(GKE_CLUSTER_NAME) $(GCP_LOCATION_FLAG) $(GKE_CLUSTER_SHAPE_FLAGS) $(GKE_FUTURE_COMPAT_FLAGS) $(GKE_CLUSTER_FLAGS) \
		--enable-pod-security-policy \
		--cluster-version $(GKE_VERSION) \
		--image-type cos_containerd \
		--tags open-match
	$(MAKE) create-cluster-role-binding
	

delete-gke-cluster: gcloud
	-$(GCLOUD) $(GCP_PROJECT_FLAG) container clusters delete $(GKE_CLUSTER_NAME) $(GCP_LOCATION_FLAG) $(GCLOUD_EXTRA_FLAGS)

create-mini-cluster: build/toolchain/bin/minikube$(EXE_EXTENSION)
	$(MINIKUBE) start --memory 6144 --cpus 4 --disk-size 50g

delete-mini-cluster: build/toolchain/bin/minikube$(EXE_EXTENSION)
	-$(MINIKUBE) delete

gcp-apply-binauthz-policy: build/policies/binauthz.yaml
	$(GCLOUD) beta $(GCP_PROJECT_FLAG) container binauthz policy import build/policies/binauthz.yaml

## ##############################
## # Protobuf
##

## # Build all protobuf definitions.
## make all-protos
##
all-protos: $(ALL_PROTOS)

# The proto generator really wants to be run from the $GOPATH root, and doesn't
# support methods for directing it to the correct location that's not the proto
# file's location. 
# So, instead, put it in a tempororary directory, then move it out.
pkg/pb/%.pb.go: api/%.proto third_party/ build/toolchain/bin/protoc$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-go$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-grpc-gateway$(EXE_EXTENSION)
	mkdir -p $(REPOSITORY_ROOT)/build/prototmp $(REPOSITORY_ROOT)/pkg/pb
	$(PROTOC) $< \
		-I $(REPOSITORY_ROOT) -I $(PROTOC_INCLUDES) \
		--go_out=plugins=grpc:$(REPOSITORY_ROOT)/build/prototmp
	mv $(REPOSITORY_ROOT)/build/prototmp/open-match.dev/open-match/$@ $@

internal/ipb/%.pb.go: internal/api/%.proto third_party/ build/toolchain/bin/protoc$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-go$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-grpc-gateway$(EXE_EXTENSION)
	mkdir -p $(REPOSITORY_ROOT)/build/prototmp $(REPOSITORY_ROOT)/internal/ipb
	$(PROTOC) $< \
		-I $(REPOSITORY_ROOT) -I $(PROTOC_INCLUDES) \
		--go_out=plugins=grpc:$(REPOSITORY_ROOT)/build/prototmp
	mv $(REPOSITORY_ROOT)/build/prototmp/open-match.dev/open-match/$@ $@

pkg/pb/%.pb.gw.go: api/%.proto third_party/ build/toolchain/bin/protoc$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-go$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-grpc-gateway$(EXE_EXTENSION)
	mkdir -p $(REPOSITORY_ROOT)/build/prototmp $(REPOSITORY_ROOT)/pkg/pb
	$(PROTOC) $< \
		-I $(REPOSITORY_ROOT) -I $(PROTOC_INCLUDES) \
   		--grpc-gateway_out=logtostderr=true,allow_delete_body=true:$(REPOSITORY_ROOT)/build/prototmp
	mv $(REPOSITORY_ROOT)/build/prototmp/open-match.dev/open-match/$@ $@

api/%.swagger.json: api/%.proto third_party/ build/toolchain/bin/protoc$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-swagger$(EXE_EXTENSION)
	$(PROTOC) $< \
		-I $(REPOSITORY_ROOT) -I $(PROTOC_INCLUDES) \
		--swagger_out=logtostderr=true,allow_delete_body=true:$(REPOSITORY_ROOT)

api/api.md: third_party/ build/toolchain/bin/protoc-gen-doc$(EXE_EXTENSION)
	$(PROTOC) api/*.proto \
		-I $(REPOSITORY_ROOT) -I $(PROTOC_INCLUDES) \
  		--doc_out=. \
  		--doc_opt=markdown,api.md
# Crazy hack that insert hugo link reference to this API doc -)
	$(SED_REPLACE) '1 i\---\
title: "Open Match API References" \
linkTitle: "Open Match API References" \
weight: 2 \
description: \
  This document provides API references for Open Match services. \
--- \
' ./api.md && mv ./api.md $(REPOSITORY_ROOT)/../open-match-docs/site/content/en/docs/Reference/

# Include structure of the protos needs to be called out do the dependency chain is run through properly.
pkg/pb/backend.pb.go: pkg/pb/messages.pb.go
pkg/pb/frontend.pb.go: pkg/pb/messages.pb.go
pkg/pb/matchfunction.pb.go: pkg/pb/messages.pb.go
pkg/pb/query.pb.go: pkg/pb/messages.pb.go
pkg/pb/evaluator.pb.go: pkg/pb/messages.pb.go
internal/ipb/synchronizer.pb.go: pkg/pb/messages.pb.go
internal/ipb/internal.pb.go: pkg/pb/messages.pb.go

build: assets
	$(GO) build ./...
	$(GO) build -tags e2ecluster ./...

define test_folder
	$(if $(wildcard $(1)/go.mod), \
		cd $(1) && \
		$(GO) test -cover -test.count $(GOLANG_TEST_COUNT) -race ./... && \
		$(GO) test -cover -test.count $(GOLANG_TEST_COUNT) -run IgnoreRace$$ ./... \
    )
	$(foreach dir, $(wildcard $(1)/*/.), $(call test_folder, $(dir)))
endef

define fast_test_folder
	$(if $(wildcard $(1)/go.mod), \
		cd $(1) && \
		$(GO) test ./... \
    )
	$(foreach dir, $(wildcard $(1)/*/.), $(call fast_test_folder, $(dir)))
endef

## # Run go tests
## make test
##
test: $(ALL_PROTOS) tls-certs third_party/
	$(call test_folder,.)

## # Run go tests more quickly, but with worse flake and race detection
## make fasttest
##
fasttest: $(ALL_PROTOS) tls-certs third_party/
	$(call fast_test_folder,.)

test-e2e-cluster: all-protos tls-certs third_party/
	$(HELM) test --timeout 7m30s -v 0 --logs -n $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(OPEN_MATCH_HELM_NAME)

fmt:
	$(GO) fmt ./...
	gofmt -s -w .

vet:
	$(GO) vet ./...

golangci: build/toolchain/bin/golangci-lint$(EXE_EXTENSION)
	GO111MODULE=on $(GOLANGCI) run --config=$(REPOSITORY_ROOT)/.golangci.yaml

## # Run linter on Go code, charts and terraform
## make lint
##
lint: fmt vet golangci lint-chart terraform-lint

assets: $(ALL_PROTOS) tls-certs third_party/ build/chart/

build/cmd: $(foreach CMD,$(CMDS),build/cmd/$(CMD))

# Building a given build/cmd folder is split into two pieces: BUILD_PHONY and
# COPY_PHONY.  The BUILD_PHONY is the common go build command, which is
# reusable.  The COPY_PHONY is used by some targets which require additional
# files to be included in the image.
$(foreach CMD,$(CMDS),build/cmd/$(CMD)): build/cmd/%: build/cmd/%/BUILD_PHONY build/cmd/%/COPY_PHONY

build/cmd/%/BUILD_PHONY:
	mkdir -p $(BUILD_DIR)/cmd/$*
	CGO_ENABLED=0 $(GO) build -a -installsuffix cgo -o $(BUILD_DIR)/cmd/$*/run open-match.dev/open-match/cmd/$*

# Default is that nothing needs to be copied into the direcotry
build/cmd/%/COPY_PHONY:
	#

build/cmd/swaggerui/COPY_PHONY:
	mkdir -p $(BUILD_DIR)/cmd/swaggerui/static/api
	cp third_party/swaggerui/* $(BUILD_DIR)/cmd/swaggerui/static/
	$(SED_REPLACE) 's|https://open-match.dev/api/v.*/|/api/|g' $(BUILD_DIR)/cmd/swaggerui/static/config.json
	cp api/*.json $(BUILD_DIR)/cmd/swaggerui/static/api/

build/cmd/demo-%/COPY_PHONY:
	mkdir -p $(BUILD_DIR)/cmd/demo-$*/
	cp -r examples/demo/static $(BUILD_DIR)/cmd/demo-$*/static

build/policies/binauthz.yaml: install/policies/binauthz.yaml
	mkdir -p $(BUILD_DIR)/policies
	cp -f $(REPOSITORY_ROOT)/install/policies/binauthz.yaml $(BUILD_DIR)/policies/binauthz.yaml
	$(SED_REPLACE) 's/$$PROJECT_ID/$(GCP_PROJECT_ID)/g' $(BUILD_DIR)/policies/binauthz.yaml
	$(SED_REPLACE) 's/$$GKE_CLUSTER_NAME/$(GKE_CLUSTER_NAME)/g' $(BUILD_DIR)/policies/binauthz.yaml
	$(SED_REPLACE) 's/$$GCP_LOCATION/$(GCP_LOCATION)/g' $(BUILD_DIR)/policies/binauthz.yaml
ifeq ($(ENABLE_SECURITY_HARDENING),1)
	$(SED_REPLACE) 's/$$EVALUATION_MODE/ALWAYS_DENY/g' $(BUILD_DIR)/policies/binauthz.yaml
else
	$(SED_REPLACE) 's/$$EVALUATION_MODE/ALWAYS_ALLOW/g' $(BUILD_DIR)/policies/binauthz.yaml
endif

terraform-test: install/terraform/open-match/.terraform/ install/terraform/open-match-build/.terraform/
	(cd $(REPOSITORY_ROOT)/install/terraform/open-match/ && $(TERRAFORM) validate)
	(cd $(REPOSITORY_ROOT)/install/terraform/open-match-build/ && $(TERRAFORM) validate)

terraform-plan: install/terraform/open-match/.terraform/
	(cd $(REPOSITORY_ROOT)/install/terraform/open-match/ && $(TERRAFORM) plan -var gcp_project_id=$(GCP_PROJECT_ID) -var gcp_location=$(GCP_LOCATION))

terraform-lint: build/toolchain/bin/terraform$(EXE_EXTENSION)
	$(TERRAFORM) fmt -recursive

terraform-apply: install/terraform/open-match/.terraform/
	(cd $(REPOSITORY_ROOT)/install/terraform/open-match/ && $(TERRAFORM) apply -var gcp_project_id=$(GCP_PROJECT_ID) -var gcp_location=$(GCP_LOCATION))

install/terraform/open-match/.terraform/: build/toolchain/bin/terraform$(EXE_EXTENSION)
	(cd $(REPOSITORY_ROOT)/install/terraform/open-match/ && $(TERRAFORM) init)

install/terraform/open-match-build/.terraform/: build/toolchain/bin/terraform$(EXE_EXTENSION)
	(cd $(REPOSITORY_ROOT)/install/terraform/open-match-build/ && $(TERRAFORM) init)

build/certificates/: build/toolchain/bin/certgen$(EXE_EXTENSION)
	mkdir -p $(BUILD_DIR)/certificates/
	cd $(BUILD_DIR)/certificates/ && $(CERTGEN)

md-test: docker
	docker run -t --rm -v $(REPOSITORY_ROOT):/mnt:ro dkhamsing/awesome_bot --white-list "localhost,https://goreportcard.com,github.com/googleforgames/open-match/tree/release-,github.com/googleforgames/open-match/blob/release-,github.com/googleforgames/open-match/releases/download/v,https://swagger.io/tools/swagger-codegen/" --allow-dupe --allow-redirect --skip-save-results `find . -type f -name '*.md' -not -path './build/*' -not -path './.git*'`

ci-deploy-artifacts: install/yaml/ $(SWAGGER_JSON_DOCS) build/chart/ gcloud
ifeq ($(_GCB_POST_SUBMIT),1)
	gsutil cp -a public-read $(REPOSITORY_ROOT)/install/yaml/* gs://open-match-chart/install/v$(BASE_VERSION)/yaml/
	gsutil cp -a public-read $(REPOSITORY_ROOT)/api/*.json gs://open-match-chart/api/v$(BASE_VERSION)/
	# Deploy Helm Chart
	# Since each build will refresh just it's version we can allow this for every post submit.
	# Copy the files into multiple locations to keep a backup.
	gsutil cp -a public-read $(BUILD_DIR)/chart/*.* gs://open-match-chart/chart/by-hash/$(VERSION)/
	gsutil cp -a public-read $(BUILD_DIR)/chart/*.* gs://open-match-chart/chart/
else
	@echo "Not deploying build artifacts to open-match.dev because this is not a post commit change."
endif

ci-reap-namespaces: build/toolchain/bin/reaper$(EXE_EXTENSION)
	-$(TOOLCHAIN_BIN)/reaper -age=30m

# For presubmit we want to update the protobuf generated files and verify that tests are good.
presubmit: GOLANG_TEST_COUNT = 5
presubmit: clean third_party/ update-chart-deps assets update-deps lint build test md-test terraform-test

build/release/: presubmit clean-install-yaml install/yaml/
	mkdir -p $(BUILD_DIR)/release/
	cp $(REPOSITORY_ROOT)/install/yaml/* $(BUILD_DIR)/release/

validate-preview-release:
ifneq ($(_GCB_POST_SUBMIT),1)
	@echo "You must run make with _GCB_POST_SUBMIT=1"
	exit 1
endif
ifneq (,$(findstring -preview,$(BASE_VERSION)))
	@echo "Creating preview for $(BASE_VERSION)"
else
	@echo "BASE_VERSION must contain -preview, it is $(BASE_VERSION)"
	exit 1
endif

preview-release: REGISTRY = gcr.io/$(OPEN_MATCH_PUBLIC_IMAGES_PROJECT_ID)
preview-release: TAG = $(BASE_VERSION)
preview-release: validate-preview-release build/release/ retag-images ci-deploy-artifacts

release: REGISTRY = gcr.io/$(OPEN_MATCH_PUBLIC_IMAGES_PROJECT_ID)
release: TAG = $(BASE_VERSION)
release: presubmit build/release/

clean-secrets:
	rm -rf $(OPEN_MATCH_SECRETS_DIR)

clean-protos:
	rm -rf $(REPOSITORY_ROOT)/build/prototmp/
	rm -rf $(REPOSITORY_ROOT)/pkg/pb/
	rm -rf $(REPOSITORY_ROOT)/internal/ipb/

clean-terraform:
	rm -rf $(REPOSITORY_ROOT)/install/terraform/.terraform/

clean-build: clean-toolchain clean-release clean-chart
	rm -rf $(BUILD_DIR)/

clean-release:
	rm -rf $(BUILD_DIR)/release/

clean-toolchain:
	rm -rf $(TOOLCHAIN_DIR)/

clean-chart:
	rm -rf $(BUILD_DIR)/chart/

clean-install-yaml:
	rm -f $(REPOSITORY_ROOT)/install/yaml/*

clean-swagger-docs:
	rm -rf $(REPOSITORY_ROOT)/api/*.json

clean-third-party:
	rm -rf $(REPOSITORY_ROOT)/third_party/

clean: clean-images clean-build clean-install-yaml clean-secrets clean-terraform clean-third-party clean-protos clean-swagger-docs

proxy-frontend: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	@echo "Frontend Health: http://localhost:$(FRONTEND_PORT)/healthz"
	@echo "Frontend RPC: http://localhost:$(FRONTEND_PORT)/debug/rpcz"
	@echo "Frontend Trace: http://localhost:$(FRONTEND_PORT)/debug/tracez"
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=open-match,component=frontend,release=$(OPEN_MATCH_HELM_NAME)" --output jsonpath='{.items[0].metadata.name}') $(FRONTEND_PORT):51504 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-backend: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	@echo "Backend Health: http://localhost:$(BACKEND_PORT)/healthz"
	@echo "Backend RPC: http://localhost:$(BACKEND_PORT)/debug/rpcz"
	@echo "Backend Trace: http://localhost:$(BACKEND_PORT)/debug/tracez"
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=open-match,component=backend,release=$(OPEN_MATCH_HELM_NAME)" --output jsonpath='{.items[0].metadata.name}') $(BACKEND_PORT):51505 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-query: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	@echo "QueryService Health: http://localhost:$(QUERY_PORT)/healthz"
	@echo "QueryService RPC: http://localhost:$(QUERY_PORT)/debug/rpcz"
	@echo "QueryService Trace: http://localhost:$(QUERY_PORT)/debug/tracez"
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=open-match,component=query,release=$(OPEN_MATCH_HELM_NAME)" --output jsonpath='{.items[0].metadata.name}') $(QUERY_PORT):51503 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-synchronizer: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	@echo "Synchronizer Health: http://localhost:$(SYNCHRONIZER_PORT)/healthz"
	@echo "Synchronizer RPC: http://localhost:$(SYNCHRONIZER_PORT)/debug/rpcz"
	@echo "Synchronizer Trace: http://localhost:$(SYNCHRONIZER_PORT)/debug/tracez"
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=open-match,component=synchronizer,release=$(OPEN_MATCH_HELM_NAME)" --output jsonpath='{.items[0].metadata.name}') $(SYNCHRONIZER_PORT):51506 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-jaeger: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	@echo "Jaeger Query Frontend: http://localhost:16686"
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app.kubernetes.io/name=jaeger,app.kubernetes.io/component=query" --output jsonpath='{.items[0].metadata.name}') $(JAEGER_QUERY_PORT):16686 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-grafana: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	@echo "User: admin"
	@echo "Password: openmatch"
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=grafana,release=$(OPEN_MATCH_HELM_NAME)" --output jsonpath='{.items[0].metadata.name}') $(GRAFANA_PORT):3000 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-prometheus: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=prometheus,component=server,release=$(OPEN_MATCH_HELM_NAME)" --output jsonpath='{.items[0].metadata.name}') $(PROMETHEUS_PORT):9090 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-dashboard: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	$(KUBECTL) port-forward --namespace kube-system $(shell $(KUBECTL) get pod --namespace kube-system --selector="app=kubernetes-dashboard" --output jsonpath='{.items[0].metadata.name}') $(DASHBOARD_PORT):9092 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-ui: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	@echo "SwaggerUI Health: http://localhost:$(SWAGGERUI_PORT)/"
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=open-match,component=swaggerui,release=$(OPEN_MATCH_HELM_NAME)" --output jsonpath='{.items[0].metadata.name}') $(SWAGGERUI_PORT):51500 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-demo: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	@echo "View Demo: http://localhost:$(DEMO_PORT)"
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE)-demo $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE)-demo --selector="app=open-match-demo,component=demo" --output jsonpath='{.items[0].metadata.name}') $(DEMO_PORT):51507 $(PORT_FORWARD_ADDRESS_FLAG)

# Run `make proxy` instead to run everything at the same time.
# If you run this directly it will just run each proxy sequentially.
proxy-all: proxy-frontend proxy-backend proxy-query proxy-grafana proxy-prometheus proxy-jaeger proxy-synchronizer proxy-ui proxy-dashboard proxy-demo

proxy:
	# This is an exception case where we'll call recursive make.
	# To simplify accessing all the proxy ports we'll call `make proxy-all` with enough subprocesses to run them concurrently.
	$(MAKE) proxy-all -j20

update-deps:
	$(GO) mod tidy

third_party/: third_party/google/api third_party/protoc-gen-swagger/options third_party/swaggerui/

third_party/google/api:
	mkdir -p $(TOOLCHAIN_DIR)/googleapis-temp/
	mkdir -p $(REPOSITORY_ROOT)/third_party/google/api
	mkdir -p $(REPOSITORY_ROOT)/third_party/google/rpc
	curl -o $(TOOLCHAIN_DIR)/googleapis-temp/googleapis.zip -L https://github.com/googleapis/googleapis/archive/$(GOOGLE_APIS_VERSION).zip
	(cd $(TOOLCHAIN_DIR)/googleapis-temp/; unzip -q -o googleapis.zip)
	cp -f $(TOOLCHAIN_DIR)/googleapis-temp/googleapis-$(GOOGLE_APIS_VERSION)/google/api/*.proto $(REPOSITORY_ROOT)/third_party/google/api/
	cp -f $(TOOLCHAIN_DIR)/googleapis-temp/googleapis-$(GOOGLE_APIS_VERSION)/google/rpc/*.proto $(REPOSITORY_ROOT)/third_party/google/rpc/
	rm -rf $(TOOLCHAIN_DIR)/googleapis-temp

third_party/protoc-gen-swagger/options:
	mkdir -p $(TOOLCHAIN_DIR)/grpc-gateway-temp/
	mkdir -p $(REPOSITORY_ROOT)/third_party/protoc-gen-swagger/options
	curl -o $(TOOLCHAIN_DIR)/grpc-gateway-temp/grpc-gateway.zip -L https://github.com/grpc-ecosystem/grpc-gateway/archive/v$(GRPC_GATEWAY_VERSION).zip
	(cd $(TOOLCHAIN_DIR)/grpc-gateway-temp/; unzip -q -o grpc-gateway.zip)
	cp -f $(TOOLCHAIN_DIR)/grpc-gateway-temp/grpc-gateway-$(GRPC_GATEWAY_VERSION)/protoc-gen-swagger/options/*.proto $(REPOSITORY_ROOT)/third_party/protoc-gen-swagger/options/
	rm -rf $(TOOLCHAIN_DIR)/grpc-gateway-temp

third_party/swaggerui/:
	mkdir -p $(TOOLCHAIN_DIR)/swaggerui-temp/
	mkdir -p $(TOOLCHAIN_BIN)
	curl -o $(TOOLCHAIN_DIR)/swaggerui-temp/swaggerui.zip -L \
		https://github.com/swagger-api/swagger-ui/archive/v$(SWAGGERUI_VERSION).zip
	(cd $(TOOLCHAIN_DIR)/swaggerui-temp/; unzip -q -o swaggerui.zip)
	cp -rf $(TOOLCHAIN_DIR)/swaggerui-temp/swagger-ui-$(SWAGGERUI_VERSION)/dist/ \
		$(REPOSITORY_ROOT)/third_party/swaggerui
	# Update the URL in the main page to point to a known good endpoint.
	cp $(REPOSITORY_ROOT)/cmd/swaggerui/config.json $(REPOSITORY_ROOT)/third_party/swaggerui/
	$(SED_REPLACE) 's|url:.*|configUrl: "/config.json",|g' $(REPOSITORY_ROOT)/third_party/swaggerui/index.html
	$(SED_REPLACE) 's|0.0.0-dev|$(BASE_VERSION)|g' $(REPOSITORY_ROOT)/third_party/swaggerui/config.json
	rm -rf $(TOOLCHAIN_DIR)/swaggerui-temp

sync-deps:
	$(GO) clean -modcache
	$(GO) mod download

# Prevents users from running with sudo.
# There's an exception for Google Cloud Build because it runs as root.
no-sudo:
ifndef OPEN_MATCH_CI_MODE
ifeq ($(shell whoami),root)
	@echo "ERROR: Running Makefile as root (or sudo)"
	@echo "Please follow the instructions at https://docs.docker.com/install/linux/linux-postinstall/ if you are trying to sudo run the Makefile because of the 'Cannot connect to the Docker daemon' error."
	@echo "NOTE: sudo/root do not have the authentication token to talk to any GCP service via gcloud."
	exit 1
endif
endif

.PHONY: docker gcloud update-deps sync-deps all build proxy-dashboard proxy-prometheus proxy-grafana clean clean-build clean-toolchain clean-binaries clean-protos presubmit test ci-reap-namespaces md-test vet

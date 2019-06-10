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
## Create a GKE Cluster (requires gcloud installed and initialized, https://cloud.google.com/sdk/docs/quickstarts)
## make enable-gcp-apis
## make create-gke-cluster push-helm
##
## Create a Minikube Cluster (requires VirtualBox)
## make create-mini-cluster push-helm
##
## Create a KinD Cluster (Follow instructions to run command before pushing helm.)
## make create-kind-cluster get-kind-kubeconfig
## Finish KinD setup by installing helm:
## make push-helm
##
## Deploy Open Match
## make push-images -j$(nproc)
## make install-chart
##
## Build and Test
## make all -j$(nproc)
## make test
##
## Access monitoring
## make proxy-prometheus
## make proxy-grafana
## make proxy-ui
##
## Teardown
## make delete-mini-cluster
## make delete-gke-cluster
## make delete-kind-cluster && export KUBECONFIG=""
##
## Prepare a Pull Request
## make presubmit

# If you want information on how to edit this file checkout,
# http://makefiletutorial.com/

BASE_VERSION = 0.0.0-dev
SHORT_SHA = $(shell git rev-parse --short=7 HEAD | tr -d [:punct:])
VERSION_SUFFIX = $(SHORT_SHA)
BRANCH_NAME = $(shell git rev-parse --abbrev-ref HEAD | tr -d [:punct:])
VERSION = $(BASE_VERSION)-$(VERSION_SUFFIX)
BUILD_DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
YEAR_MONTH = $(shell date -u +'%Y%m')
MAJOR_MINOR_VERSION = $(shell echo $(BASE_VERSION) | cut -d '.' -f1).$(shell echo $(BASE_VERSION) | cut -d '.' -f2)

PROTOC_VERSION = 3.7.1
HELM_VERSION = 2.14.0
HUGO_VERSION = 0.55.5
KUBECTL_VERSION = 1.14.2
NODEJS_VERSION = 10.15.3
SKAFFOLD_VERSION = latest
MINIKUBE_VERSION = latest
HTMLTEST_VERSION = 0.10.3
GOLANGCI_VERSION = 1.16.0
KIND_VERSION = 0.3.0
SWAGGERUI_VERSION = 3.22.2

ENABLE_SECURITY_HARDENING = 0
GO = GO111MODULE=on go
# Defines the absolute local directory of the open-match project
REPOSITORY_ROOT := $(patsubst %/,%,$(dir $(abspath $(MAKEFILE_LIST))))
GO_BUILD_COMMAND = CGO_ENABLED=0 $(GO) build -a -installsuffix cgo .
BUILD_DIR = $(REPOSITORY_ROOT)/build
TOOLCHAIN_DIR = $(BUILD_DIR)/toolchain
TOOLCHAIN_BIN = $(TOOLCHAIN_DIR)/bin
PROTOC := $(TOOLCHAIN_BIN)/protoc
PROTOC_INCLUDES := $(REPOSITORY_ROOT)/third_party
GCP_PROJECT_ID ?=
GCP_PROJECT_FLAG = --project=$(GCP_PROJECT_ID)
OPEN_MATCH_PUBLIC_IMAGES_PROJECT_ID = open-match-public-images
OM_SITE_GCP_PROJECT_ID = open-match-site
OM_SITE_GCP_PROJECT_FLAG = --project=$(OM_SITE_GCP_PROJECT_ID)
REGISTRY ?= gcr.io/$(GCP_PROJECT_ID)
TAG := $(VERSION)
ALTERNATE_TAG := dev
GKE_CLUSTER_NAME = om-cluster
GCP_REGION = us-west1
GCP_ZONE = us-west1-a
EXE_EXTENSION =
GCP_LOCATION_FLAG = --zone $(GCP_ZONE)
GO111MODULE = on
SWAGGERUI_PORT = 51500
PROMETHEUS_PORT = 9090
GRAFANA_PORT = 3000
SITE_PORT = 8080
FRONTEND_PORT = 51504
BACKEND_PORT = 51505
MMLOGIC_PORT = 51503
EVALUATOR_PORT = 51506
HELM = $(TOOLCHAIN_BIN)/helm
TILLER = $(TOOLCHAIN_BIN)/tiller
MINIKUBE = $(TOOLCHAIN_BIN)/minikube
KUBECTL = $(TOOLCHAIN_BIN)/kubectl
HTMLTEST = $(TOOLCHAIN_BIN)/htmltest
KIND = $(TOOLCHAIN_BIN)/kind
OPEN_MATCH_CHART_NAME = open-match
OPEN_MATCH_KUBERNETES_NAMESPACE = open-match
OPEN_MATCH_DEMO_CHART_NAME = open-match-demo
OPEN_MATCH_DEMO_KUBERNETES_NAMESPACE = open-match
OPEN_MATCH_SECRETS_DIR = $(REPOSITORY_ROOT)/install/helm/open-match/secrets
REDIS_NAME = om-redis
GCLOUD_ACCOUNT_EMAIL = $(shell gcloud auth list --format yaml | grep account: | cut -c 10-)
_GCB_POST_SUBMIT ?= 0
# Latest version triggers builds of :latest images and deploy to main website.
_GCB_LATEST_VERSION ?= undefined
IMAGE_BUILD_ARGS=--build-arg BUILD_DATE=$(BUILD_DATE) --build-arg=VCS_REF=$(SHORT_SHA) --build-arg BUILD_VERSION=$(BASE_VERSION)

# Make port forwards accessible outside of the proxy machine.
PORT_FORWARD_ADDRESS_FLAG = --address 0.0.0.0
DASHBOARD_PORT = 9092

# AppEngine variables
GAE_SITE_VERSION = om$(YEAR_MONTH)

# If the version is 0.0* then the service name is "development" as in development.open-match.dev.
ifeq ($(MAJOR_MINOR_VERSION),0.0)
	GAE_SERVICE_NAME = development
else
	GAE_SERVICE_NAME = $(shell echo $(MAJOR_MINOR_VERSION) | tr . -)
endif

export PATH := $(REPOSITORY_ROOT)/node_modules/.bin/:$(TOOLCHAIN_BIN):$(TOOLCHAIN_DIR)/nodejs/bin:$(PATH)

# Get the project from gcloud if it's not set.
ifeq ($(GCP_PROJECT_ID),)
	export GCP_PROJECT_ID = $(shell gcloud config list --format 'value(core.project)')
endif

ifeq ($(OS),Windows_NT)
	# TODO: Windows packages are here but things are broken since many paths are Linux based and zip vs tar.gz.
	HELM_PACKAGE = https://storage.googleapis.com/kubernetes-helm/helm-v$(HELM_VERSION)-windows-amd64.zip
	MINIKUBE_PACKAGE = https://storage.googleapis.com/minikube/releases/$(MINIKUBE_VERSION)/minikube-windows-amd64.exe
	SKAFFOLD_PACKAGE = https://storage.googleapis.com/skaffold/releases/$(SKAFFOLD_VERSION)/skaffold-windows-amd64.exe
	EXE_EXTENSION = .exe
	PROTOC_PACKAGE = https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-win64.zip
	KUBECTL_PACKAGE = https://storage.googleapis.com/kubernetes-release/release/v$(KUBECTL_VERSION)/bin/windows/amd64/kubectl.exe
	HUGO_PACKAGE = https://github.com/gohugoio/hugo/releases/download/v$(HUGO_VERSION)/hugo_extended_$(HUGO_VERSION)_Windows-64bit.zip
	NODEJS_PACKAGE = https://nodejs.org/dist/v$(NODEJS_VERSION)/node-v$(NODEJS_VERSION)-win-x64.zip
	NODEJS_PACKAGE_NAME = nodejs.zip
	HTMLTEST_PACKAGE = https://github.com/wjdp/htmltest/releases/download/v$(HTMLTEST_VERSION)/htmltest_$(HTMLTEST_VERSION)_windows_amd64.zip
	GOLANGCI_PACKAGE = https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_VERSION)/golangci-lint-$(GOLANGCI_VERSION)-windows-amd64.zip
	KIND_PACKAGE = https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-windows-amd64
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		HELM_PACKAGE = https://storage.googleapis.com/kubernetes-helm/helm-v$(HELM_VERSION)-linux-amd64.tar.gz
		MINIKUBE_PACKAGE = https://storage.googleapis.com/minikube/releases/$(MINIKUBE_VERSION)/minikube-linux-amd64
		SKAFFOLD_PACKAGE = https://storage.googleapis.com/skaffold/releases/$(SKAFFOLD_VERSION)/skaffold-linux-amd64
		PROTOC_PACKAGE = https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-linux-x86_64.zip
		KUBECTL_PACKAGE = https://storage.googleapis.com/kubernetes-release/release/v$(KUBECTL_VERSION)/bin/linux/amd64/kubectl
		HUGO_PACKAGE = https://github.com/gohugoio/hugo/releases/download/v$(HUGO_VERSION)/hugo_extended_$(HUGO_VERSION)_Linux-64bit.tar.gz
		NODEJS_PACKAGE = https://nodejs.org/dist/v$(NODEJS_VERSION)/node-v$(NODEJS_VERSION)-linux-x64.tar.gz
		NODEJS_PACKAGE_NAME = nodejs.tar.gz
		HTMLTEST_PACKAGE = https://github.com/wjdp/htmltest/releases/download/v$(HTMLTEST_VERSION)/htmltest_$(HTMLTEST_VERSION)_linux_amd64.tar.gz
		GOLANGCI_PACKAGE = https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_VERSION)/golangci-lint-$(GOLANGCI_VERSION)-linux-amd64.tar.gz
		KIND_PACKAGE = https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-linux-amd64
	endif
	ifeq ($(UNAME_S),Darwin)
		HELM_PACKAGE = https://storage.googleapis.com/kubernetes-helm/helm-v$(HELM_VERSION)-darwin-amd64.tar.gz
		MINIKUBE_PACKAGE = https://storage.googleapis.com/minikube/releases/$(MINIKUBE_VERSION)/minikube-darwin-amd64
		SKAFFOLD_PACKAGE = https://storage.googleapis.com/skaffold/releases/$(SKAFFOLD_VERSION)/skaffold-darwin-amd64
		PROTOC_PACKAGE = https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)-osx-x86_64.zip
		KUBECTL_PACKAGE = https://storage.googleapis.com/kubernetes-release/release/v$(KUBECTL_VERSION)/bin/darwin/amd64/kubectl
		HUGO_PACKAGE = https://github.com/gohugoio/hugo/releases/download/v$(HUGO_VERSION)/hugo_extended_$(HUGO_VERSION)_macOS-64bit.tar.gz
		NODEJS_PACKAGE = https://nodejs.org/dist/v$(NODEJS_VERSION)/node-v$(NODEJS_VERSION)-darwin-x64.tar.gz
		NODEJS_PACKAGE_NAME = nodejs.tar.gz
		HTMLTEST_PACKAGE = https://github.com/wjdp/htmltest/releases/download/v$(HTMLTEST_VERSION)/htmltest_$(HTMLTEST_VERSION)_osx_amd64.tar.gz
		GOLANGCI_PACKAGE = https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_VERSION)/golangci-lint-$(GOLANGCI_VERSION)-darwin-amd64.tar.gz
		KIND_PACKAGE = https://github.com/kubernetes-sigs/kind/releases/download/v$(KIND_VERSION)/kind-darwin-amd64
	endif
endif

help:
	@cat Makefile | grep ^\#\# | grep -v ^\#\#\# |cut -c 4-

local-cloud-build: LOCAL_CLOUD_BUILD_PUSH = # --push
local-cloud-build: gcloud
	cloud-build-local --config=cloudbuild.yaml --dryrun=false $(LOCAL_CLOUD_BUILD_PUSH) --substitutions SHORT_SHA=$(VERSION_SUFFIX),_GCB_POST_SUBMIT=$(_GCB_POST_SUBMIT),_GCB_LATEST_VERSION=$(_GCB_LATEST_VERSION),BRANCH_NAME=$(BRANCH_NAME) .

push-images: push-service-images push-example-images

push-service-images: push-backend-image push-frontend-image push-mmlogic-image push-minimatch-image push-evaluator-image push-swaggerui-image

push-backend-image: docker build-backend-image
	docker push $(REGISTRY)/openmatch-backend:$(TAG)
	docker push $(REGISTRY)/openmatch-backend:$(ALTERNATE_TAG)

push-frontend-image: docker build-frontend-image
	docker push $(REGISTRY)/openmatch-frontend:$(TAG)
	docker push $(REGISTRY)/openmatch-frontend:$(ALTERNATE_TAG)

push-mmlogic-image: docker build-mmlogic-image
	docker push $(REGISTRY)/openmatch-mmlogic:$(TAG)
	docker push $(REGISTRY)/openmatch-mmlogic:$(ALTERNATE_TAG)

push-minimatch-image: docker build-minimatch-image
	docker push $(REGISTRY)/openmatch-minimatch:$(TAG)
	docker push $(REGISTRY)/openmatch-minimatch:$(ALTERNATE_TAG)

push-evaluator-image: docker build-evaluator-image
	docker push $(REGISTRY)/openmatch-evaluator:$(TAG)
	docker push $(REGISTRY)/openmatch-evaluator:$(ALTERNATE_TAG)

push-swaggerui-image: docker build-swaggerui-image
	docker push $(REGISTRY)/openmatch-swaggerui:$(TAG)
	docker push $(REGISTRY)/openmatch-swaggerui:$(ALTERNATE_TAG)

push-example-images: push-demo-images push-mmf-example-images

push-demo-images: push-mmf-go-soloduel-image push-demo-image

push-demo-image: docker build-demo-image
	docker push $(REGISTRY)/openmatch-demo:$(TAG)
	docker push $(REGISTRY)/openmatch-demo:$(ALTERNATE_TAG)

push-mmf-example-images: push-mmf-go-soloduel-image

push-mmf-go-soloduel-image: docker build-mmf-go-soloduel-image
	docker push $(REGISTRY)/openmatch-mmf-go-soloduel:$(TAG)
	docker push $(REGISTRY)/openmatch-mmf-go-soloduel:$(ALTERNATE_TAG)

build-images: build-service-images build-example-images

build-service-images: build-backend-image build-frontend-image build-mmlogic-image build-minimatch-image build-evaluator-image build-swaggerui-image

# Include all-protos here so that all dependencies are guaranteed to be downloaded after the base image is created.
# This is important so that the repository does not have any mutations while building individual images.
build-base-build-image: docker all-protos
	docker build -f Dockerfile.base-build -t open-match-base-build .

build-backend-image: docker build-base-build-image
	docker build -f cmd/backend/Dockerfile $(IMAGE_BUILD_ARGS) -t $(REGISTRY)/openmatch-backend:$(TAG) -t $(REGISTRY)/openmatch-backend:$(ALTERNATE_TAG) .

build-frontend-image: docker build-base-build-image
	docker build -f cmd/frontend/Dockerfile $(IMAGE_BUILD_ARGS) -t $(REGISTRY)/openmatch-frontend:$(TAG) -t $(REGISTRY)/openmatch-frontend:$(ALTERNATE_TAG) .

build-mmlogic-image: docker build-base-build-image
	docker build -f cmd/mmlogic/Dockerfile $(IMAGE_BUILD_ARGS) -t $(REGISTRY)/openmatch-mmlogic:$(TAG) -t $(REGISTRY)/openmatch-mmlogic:$(ALTERNATE_TAG) .

build-minimatch-image: docker build-base-build-image
	docker build -f cmd/minimatch/Dockerfile $(IMAGE_BUILD_ARGS) -t $(REGISTRY)/openmatch-minimatch:$(TAG) -t $(REGISTRY)/openmatch-minimatch:$(ALTERNATE_TAG) .

build-evaluator-image: docker build-base-build-image
	docker build -f cmd/evaluator/Dockerfile $(IMAGE_BUILD_ARGS) -t $(REGISTRY)/openmatch-evaluator:$(TAG) -t $(REGISTRY)/openmatch-evaluator:$(ALTERNATE_TAG) .

build-swaggerui-image: docker build-base-build-image site/static/swaggerui/
	docker build -f cmd/swaggerui/Dockerfile $(IMAGE_BUILD_ARGS) -t $(REGISTRY)/openmatch-swaggerui:$(TAG) -t $(REGISTRY)/openmatch-swaggerui:$(ALTERNATE_TAG) .

build-example-images: build-demo-images build-mmf-example-images

build-demo-images: build-mmf-go-soloduel-image build-demo-image

build-demo-image: docker build-base-build-image
  docker build -f examples/demo/Dockerfile -t $(REGISTRY)/openmatch-demo:$(TAG) -t $(REGISTRY)/openmatch-demo:$(ALTERNATE_TAG) .

build-mmf-example-images: build-mmf-go-soloduel-image

build-mmf-go-soloduel-image: docker build-base-build-image
	docker build -f examples/functions/golang/soloduel/Dockerfile -t $(REGISTRY)/openmatch-mmf-go-soloduel:$(TAG) -t $(REGISTRY)/openmatch-mmf-go-soloduel:$(ALTERNATE_TAG) .

clean-images: docker
	-docker rmi -f open-match-base-build
	-docker rmi -f $(REGISTRY)/openmatch-mmf-go-soloduel:$(TAG) $(REGISTRY)/openmatch-mmf-go-soloduel:$(ALTERNATE_TAG)
	-docker rmi -f $(REGISTRY)/openmatch-backend:$(TAG) $(REGISTRY)/openmatch-backend:$(ALTERNATE_TAG)
	-docker rmi -f $(REGISTRY)/openmatch-frontend:$(TAG) $(REGISTRY)/openmatch-frontend:$(ALTERNATE_TAG)
	-docker rmi -f $(REGISTRY)/openmatch-mmlogic:$(TAG) $(REGISTRY)/openmatch-mmlogic:$(ALTERNATE_TAG)
	-docker rmi -f $(REGISTRY)/openmatch-evaluator:$(TAG) $(REGISTRY)/openmatch-evaluator:$(ALTERNATE_TAG)
	-docker rmi -f $(REGISTRY)/openmatch-minimatch:$(TAG) $(REGISTRY)/openmatch-minimatch:$(ALTERNATE_TAG)
	-docker rmi -f $(REGISTRY)/openmatch-swaggerui:$(TAG) $(REGISTRY)/openmatch-swaggerui:$(ALTERNATE_TAG)

install-redis: build/toolchain/bin/helm$(EXE_EXTENSION)
	$(HELM) upgrade --install --wait --debug $(REDIS_NAME) stable/redis --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE)

update-chart-deps: build/toolchain/bin/helm$(EXE_EXTENSION)
	(cd install/helm/open-match; $(HELM) repo add incubator https://kubernetes-charts-incubator.storage.googleapis.com; $(HELM) dependency update)

lint-chart: build/toolchain/bin/helm$(EXE_EXTENSION)
	(cd install/helm; $(HELM) lint $(OPEN_MATCH_CHART_NAME); $(HELM) lint $(OPEN_MATCH_DEMO_CHART_NAME))

print-chart: build/toolchain/bin/helm$(EXE_EXTENSION)
	(cd install/helm; $(HELM) install --name $(OPEN_MATCH_CHART_NAME) --dry-run --debug $(OPEN_MATCH_CHART_NAME); $(HELM) install --name $(OPEN_MATCH_DEMO_CHART_NAME) --dry-run --debug $(OPEN_MATCH_DEMO_CHART_NAME))

install-chart: build/toolchain/bin/helm$(EXE_EXTENSION)
	$(HELM) upgrade $(OPEN_MATCH_CHART_NAME) --install --wait --debug install/helm/open-match \
		--timeout=400 \
		--namespace=$(OPEN_MATCH_KUBERNETES_NAMESPACE) \
		--set openmatch.image.registry=$(REGISTRY) \
		--set openmatch.image.tag=$(TAG) \
		--set grafana.enabled=true \
		--set jaeger.enabled=true \
		--set prometheus.enabled=true \
		--set redis.enabled=true \
		--set openmatch.monitoring.stackdriver.enabled=true \
		--set openmatch.monitoring.stackdriver.gcpProjectId=$(GCP_PROJECT_ID)

install-demo-chart: build/toolchain/bin/helm$(EXE_EXTENSION)
	$(HELM) upgrade $(OPEN_MATCH_DEMO_CHART_NAME) --install --wait --debug install/helm/open-match-demo \
	  --namespace=$(OPEN_MATCH_KUBERNETES_NAMESPACE) \
	  --set openmatch.image.registry=$(REGISTRY) \
	  --set openmatch.image.tag=$(TAG)

delete-demo-chart: build/toolchain/bin/helm$(EXE_EXTENSION)
	-$(HELM) delete --purge $(OPEN_MATCH_DEMO_CHART_NAME)

dry-chart: build/toolchain/bin/helm$(EXE_EXTENSION)
	$(HELM) upgrade --install --wait --debug --dry-run $(OPEN_MATCH_CHART_NAME) install/helm/open-match \
		--namespace=$(OPEN_MATCH_KUBERNETES_NAMESPACE) \
		--set openmatch.image.registry=$(REGISTRY) \
		--set openmatch.image.tag=$(TAG)

delete-chart: build/toolchain/bin/helm$(EXE_EXTENSION) build/toolchain/bin/kubectl$(EXE_EXTENSION)
	-$(HELM) delete --purge $(OPEN_MATCH_CHART_NAME)
	-$(KUBECTL) --ignore-not-found=true delete crd prometheuses.monitoring.coreos.com
	-$(KUBECTL) --ignore-not-found=true delete crd servicemonitors.monitoring.coreos.com
	-$(KUBECTL) --ignore-not-found=true delete crd prometheusrules.monitoring.coreos.com

install/yaml/: install/yaml/install.yaml install/yaml/install-demo.yaml install/yaml/01-redis-chart.yaml install/yaml/02-open-match.yaml install/yaml/03-prometheus-chart.yaml install/yaml/04-grafana-chart.yaml install/yaml/05-jaeger-chart.yaml

install/yaml/01-redis-chart.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template --name $(OPEN_MATCH_CHART_NAME) --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) \
		--set redis.fullnameOverride='$(REDIS_NAME)' \
		--set openmatch.config.install=false \
		--set openmatch.backend.install=false \
		--set openmatch.frontend.install=false \
		--set openmatch.mmlogic.install=false \
		--set openmatch.evaluator.install=false \
		--set openmatch.swaggerui.install=false \
		--set redis.enabled=true \
		--set prometheus.enabled=false \
		--set grafana.enabled=false \
		--set jaeger.enabled=false \
		install/helm/open-match > install/yaml/01-redis-chart.yaml

install/yaml/02-open-match.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template --name $(OPEN_MATCH_CHART_NAME) --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) \
		--set redis.fullnameOverride='$(REDIS_NAME)' \
		--set redis.enabled=false \
		--set prometheus.enabled=false \
		--set grafana.enabled=false \
		--set openmatch.image.registry=$(REGISTRY) \
		--set openmatch.image.tag=$(TAG) \
		--set openmatch.noChartMeta=true \
		install/helm/open-match > install/yaml/02-open-match.yaml

install/yaml/03-prometheus-chart.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template --name $(OPEN_MATCH_CHART_NAME) --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) \
		--set redis.enabled=false \
		--set openmatch.config.install=false \
		--set openmatch.backend.install=false \
		--set openmatch.frontend.install=false \
		--set openmatch.mmlogic.install=false \
		--set openmatch.evaluator.install=false \
		--set openmatch.swaggerui.install=false \
		--set redis.enabled=false \
		--set prometheus.enabled=true \
		--set grafana.enabled=false \
		--set jaeger.enabled=false \
		install/helm/open-match > install/yaml/03-prometheus-chart.yaml

install/yaml/04-grafana-chart.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template --name $(OPEN_MATCH_CHART_NAME) --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) \
		--set redis.enabled=false \
		--set openmatch.config.install=false \
		--set openmatch.backend.install=false \
		--set openmatch.frontend.install=false \
		--set openmatch.mmlogic.install=false \
		--set openmatch.evaluator.install=false \
		--set openmatch.swaggerui.install=false \
		--set redis.enabled=false \
		--set prometheus.enabled=false \
		--set grafana.enabled=true \
		--set jaeger.enabled=false \
		install/helm/open-match > install/yaml/04-grafana-chart.yaml

install/yaml/05-jaeger-chart.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template --name $(OPEN_MATCH_CHART_NAME) --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) \
		--set redis.fullnameOverride='$(REDIS_NAME)' \
		--set openmatch.config.install=false \
		--set openmatch.backend.install=false \
		--set openmatch.frontend.install=false \
		--set openmatch.mmlogic.install=false \
		--set openmatch.evaluator.install=false \
		--set openmatch.swaggerui.install=false \
		--set redis.enabled=false \
		--set prometheus.enabled=false \
		--set grafana.enabled=false \
		--set jaeger.enabled=true \
		install/helm/open-match > install/yaml/05-jaeger-chart.yaml

install/yaml/install.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template --name $(OPEN_MATCH_CHART_NAME) --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) \
		--set openmatch.image.registry=$(REGISTRY) \
		--set openmatch.image.tag=$(TAG) \
		--set redis.enabled=true \
		--set prometheus.enabled=true \
		--set grafana.enabled=true \
		--set jaeger.enabled=true \
		install/helm/open-match > install/yaml/install.yaml

install/yaml/install-demo.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template --name $(OPEN_MATCH_DEMO_CHART_NAME) --namespace $(OPEN_MATCH_DEMO_KUBERNETES_NAMESPACE) \
		--set openmatch.image.registry=$(REGISTRY) \
		--set openmatch.image.tag=$(TAG) \
		install/helm/open-match-demo > install/yaml/install-demo.yaml

set-redis-password:
	@stty -echo; \
		printf "Redis password: "; \
		read REDIS_PASSWORD; \
		stty echo; \
		printf "\n"; \
		$(KUBECTL) create secret generic $(REDIS_NAME) -n $(OPEN_MATCH_DEMO_KUBERNETES_NAMESPACE) --from-literal=redis-password=$$REDIS_PASSWORD --dry-run -o yaml | $(KUBECTL) replace -f - --force

install-toolchain: install-kubernetes-tools install-web-tools install-protoc-tools install-openmatch-tools
install-kubernetes-tools: build/toolchain/bin/kubectl$(EXE_EXTENSION) build/toolchain/bin/helm$(EXE_EXTENSION) build/toolchain/bin/minikube$(EXE_EXTENSION) build/toolchain/bin/skaffold$(EXE_EXTENSION)
install-web-tools: build/toolchain/bin/hugo$(EXE_EXTENSION) build/toolchain/bin/htmltest$(EXE_EXTENSION) build/toolchain/nodejs/
install-protoc-tools: build/toolchain/bin/protoc$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-go$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-grpc-gateway$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-swagger$(EXE_EXTENSION)
install-openmatch-tools: build/toolchain/bin/certgen$(EXE_EXTENSION)

build/toolchain/bin/helm$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	mkdir -p $(TOOLCHAIN_DIR)/temp-helm
ifeq ($(suffix $(HELM_PACKAGE)),.zip)
	cd $(TOOLCHAIN_DIR)/temp-helm && curl -Lo helm.zip $(HELM_PACKAGE) && unzip -j -q -o helm.zip
else
	cd $(TOOLCHAIN_DIR)/temp-helm && curl -Lo helm.tar.gz $(HELM_PACKAGE) && tar xzf helm.tar.gz --strip-components 1
endif
	mv $(TOOLCHAIN_DIR)/temp-helm/helm$(EXE_EXTENSION) $(TOOLCHAIN_BIN)/helm$(EXE_EXTENSION)
	mv $(TOOLCHAIN_DIR)/temp-helm/tiller$(EXE_EXTENSION) $(TOOLCHAIN_BIN)/tiller$(EXE_EXTENSION)
	rm -rf $(TOOLCHAIN_DIR)/temp-helm/

build/toolchain/bin/hugo$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	mkdir -p $(TOOLCHAIN_DIR)/temp-hugo
ifeq ($(suffix $(HUGO_PACKAGE)),.zip)
	cd $(TOOLCHAIN_DIR)/temp-hugo && curl -Lo hugo.zip $(HUGO_PACKAGE) && unzip -q -o hugo.zip
else
	cd $(TOOLCHAIN_DIR)/temp-hugo && curl -Lo hugo.tar.gz $(HUGO_PACKAGE) && tar xzf hugo.tar.gz
endif
	mv $(TOOLCHAIN_DIR)/temp-hugo/hugo$(EXE_EXTENSION) $(TOOLCHAIN_BIN)/hugo$(EXE_EXTENSION)
	rm -rf $(TOOLCHAIN_DIR)/temp-hugo/

build/toolchain/bin/minikube$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	curl -Lo minikube$(EXE_EXTENSION) $(MINIKUBE_PACKAGE)
	chmod +x minikube$(EXE_EXTENSION)
	mv minikube$(EXE_EXTENSION) $(TOOLCHAIN_BIN)/minikube$(EXE_EXTENSION)

build/toolchain/bin/kubectl$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	curl -Lo kubectl$(EXE_EXTENSION) $(KUBECTL_PACKAGE)
	chmod +x kubectl$(EXE_EXTENSION)
	mv kubectl$(EXE_EXTENSION) $(TOOLCHAIN_BIN)/kubectl$(EXE_EXTENSION)

build/toolchain/bin/skaffold$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	curl -Lo skaffold$(EXE_EXTENSION) $(SKAFFOLD_PACKAGE)
	chmod +x skaffold$(EXE_EXTENSION)
	mv skaffold$(EXE_EXTENSION) $(TOOLCHAIN_BIN)/skaffold$(EXE_EXTENSION)

build/toolchain/bin/htmltest$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	mkdir -p $(TOOLCHAIN_DIR)/temp-htmltest
ifeq ($(suffix $(HTMLTEST_PACKAGE)),.zip)
	cd $(TOOLCHAIN_DIR)/temp-htmltest && curl -Lo htmltest.zip $(HTMLTEST_PACKAGE) && unzip -q -o htmltest.zip
else
	cd $(TOOLCHAIN_DIR)/temp-htmltest && curl -Lo htmltest.tar.gz $(HTMLTEST_PACKAGE) && tar xzf htmltest.tar.gz
endif
	mv $(TOOLCHAIN_DIR)/temp-htmltest/htmltest$(EXE_EXTENSION) $(TOOLCHAIN_BIN)/htmltest$(EXE_EXTENSION)
	rm -rf $(TOOLCHAIN_DIR)/temp-htmltest/

build/toolchain/bin/golangci-lint$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	mkdir -p $(TOOLCHAIN_DIR)/temp-golangci
ifeq ($(suffix $(GOLANGCI_PACKAGE)),.zip)
	cd $(TOOLCHAIN_DIR)/temp-golangci && curl -Lo golangci.zip $(GOLANGCI_PACKAGE) && unzip -j -q -o golangci.zip
else
	cd $(TOOLCHAIN_DIR)/temp-golangci && curl -Lo golangci.tar.gz $(GOLANGCI_PACKAGE) && tar xzf golangci.tar.gz --strip-components 1
endif
	mv $(TOOLCHAIN_DIR)/temp-golangci/golangci-lint$(EXE_EXTENSION) $(TOOLCHAIN_BIN)/golangci-lint$(EXE_EXTENSION)
	rm -rf $(TOOLCHAIN_DIR)/temp-golangci/

build/toolchain/bin/kind$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	curl -Lo $(TOOLCHAIN_BIN)/kind$(EXE_EXTENSION) $(KIND_PACKAGE)
	chmod +x $(TOOLCHAIN_BIN)/kind$(EXE_EXTENSION)

build/toolchain/python/:
	virtualenv --python=python3 $(TOOLCHAIN_DIR)/python/
	# Hack to workaround some crazy bug in pip that's chopping off python executable's name.
	cd build/toolchain/python/bin && ln -s python3 pytho
	cd build/toolchain/python/ && . bin/activate && pip install locustio && deactivate

build/toolchain/bin/protoc$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	curl -o $(TOOLCHAIN_DIR)/protoc-temp.zip -L $(PROTOC_PACKAGE)
	(cd $(TOOLCHAIN_DIR); unzip -q -o protoc-temp.zip)
	rm $(TOOLCHAIN_DIR)/protoc-temp.zip $(TOOLCHAIN_DIR)/readme.txt

build/toolchain/bin/protoc-gen-go$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	cd $(TOOLCHAIN_BIN) && $(GO) build -pkgdir . github.com/golang/protobuf/protoc-gen-go

build/toolchain/bin/protoc-gen-grpc-gateway$(EXE_EXTENSION):
	cd $(TOOLCHAIN_BIN) && $(GO) build -pkgdir . github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway

build/toolchain/bin/protoc-gen-swagger$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	cd $(TOOLCHAIN_BIN) && $(GO) build -pkgdir . github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger

build/archives/$(NODEJS_PACKAGE_NAME):
	mkdir -p build/archives/
	cd build/archives/ && curl -L -o $(NODEJS_PACKAGE_NAME) $(NODEJS_PACKAGE)

build/toolchain/nodejs/: build/archives/$(NODEJS_PACKAGE_NAME)
	mkdir -p build/toolchain/nodejs/
ifeq ($(suffix $(NODEJS_PACKAGE_NAME)),.zip)
	# TODO: This is broken, there's the node-v10.15.3-win-x64 directory also windows does not have the bin/ directory.
	# https://superuser.com/questions/518347/equivalent-to-tars-strip-components-1-in-unzip
	cd build/toolchain/nodejs/ && unzip -q -o ../../archives/$(NODEJS_PACKAGE_NAME)
else
	cd build/toolchain/nodejs/ && tar xzf ../../archives/$(NODEJS_PACKAGE_NAME) --strip-components 1
endif

build/toolchain/bin/certgen$(EXE_EXTENSION): tools/certgen/certgen$(EXE_EXTENSION)
	mkdir -p $(TOOLCHAIN_BIN)
	cp -f $(REPOSITORY_ROOT)/tools/certgen/certgen$(EXE_EXTENSION) $(TOOLCHAIN_BIN)/certgen$(EXE_EXTENSION)

push-helm: build/toolchain/bin/helm$(EXE_EXTENSION) build/toolchain/bin/kubectl$(EXE_EXTENSION)
	$(KUBECTL) create serviceaccount --namespace kube-system tiller
	$(HELM) init --service-account tiller --force-upgrade
	$(KUBECTL) create clusterrolebinding tiller-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:tiller
ifneq ($(strip $($(KUBECTL) get clusterroles | grep -i rbac)),)
	$(KUBECTL) patch deploy --namespace kube-system tiller-deploy -p '{"spec":{"template":{"spec":{"serviceAccount":"tiller"}}}}'
endif
	@echo "Waiting for Tiller to become ready..."
	$(KUBECTL) wait deployment --timeout=60s --for condition=available -l app=helm,name=tiller --namespace kube-system

delete-helm: build/toolchain/bin/helm$(EXE_EXTENSION) build/toolchain/bin/kubectl$(EXE_EXTENSION)
	-$(HELM) reset
	-$(KUBECTL) --ignore-not-found=true delete serviceaccount --namespace kube-system tiller
	-$(KUBECTL) --ignore-not-found=true delete clusterrolebinding tiller-cluster-rule
ifneq ($(strip $($(KUBECTL) get clusterroles | grep -i rbac)),)
	-$(KUBECTL) --ignore-not-found=true delete deployment --namespace kube-system tiller-deploy
endif
	@echo "Waiting for Tiller to go away..."
	-$(KUBECTL) wait deployment --timeout=60s --for delete -l app=helm,name=tiller --namespace kube-system

# Fake target for docker
docker: no-sudo

# Fake target for gcloud
gcloud: no-sudo

tls-certs: install/helm/open-match/secrets/

install/helm/open-match/secrets/: install/helm/open-match/secrets/tls/root-ca/ install/helm/open-match/secrets/tls/server/

install/helm/open-match/secrets/tls/root-ca/: build/toolchain/bin/certgen$(EXE_EXTENSION)
	mkdir -p $(OPEN_MATCH_SECRETS_DIR)/tls/root-ca
	$(TOOLCHAIN_BIN)/certgen$(EXE_EXTENSION) -ca=true -publiccertificate=$(OPEN_MATCH_SECRETS_DIR)/tls/root-ca/public.cert -privatekey=$(OPEN_MATCH_SECRETS_DIR)/tls/root-ca/private.key

install/helm/open-match/secrets/tls/server/: build/toolchain/bin/certgen$(EXE_EXTENSION) install/helm/open-match/secrets/tls/root-ca/
	mkdir -p $(OPEN_MATCH_SECRETS_DIR)/tls/server/
	$(TOOLCHAIN_BIN)/certgen$(EXE_EXTENSION) -publiccertificate=$(OPEN_MATCH_SECRETS_DIR)/tls/server/public.cert -privatekey=$(OPEN_MATCH_SECRETS_DIR)/tls/server/private.key -rootpubliccertificate=$(OPEN_MATCH_SECRETS_DIR)/tls/root-ca/public.cert -rootprivatekey=$(OPEN_MATCH_SECRETS_DIR)/tls/root-ca/private.key

auth-docker: gcloud docker
	gcloud $(GCP_PROJECT_FLAG) auth configure-docker

auth-gke-cluster: gcloud
	gcloud $(GCP_PROJECT_FLAG) container clusters get-credentials $(GKE_CLUSTER_NAME) $(GCP_LOCATION_FLAG)

activate-gcp-apis: gcloud
	gcloud services enable containerregistry.googleapis.com
	gcloud services enable container.googleapis.com
	gcloud services enable containeranalysis.googleapis.com
	gcloud services enable binaryauthorization.googleapis.com

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

create-gke-cluster: GKE_VERSION = 1.13.6-gke.0 # gcloud beta container get-server-config --zone us-central1-a
create-gke-cluster: GKE_CLUSTER_SHAPE_FLAGS = --machine-type n1-standard-4 --enable-autoscaling --min-nodes 1 --num-nodes 2 --max-nodes 10 --disk-size 50
create-gke-cluster: GKE_FUTURE_COMPAT_FLAGS = --no-enable-basic-auth --no-issue-client-certificate --enable-ip-alias --metadata disable-legacy-endpoints=true --enable-autoupgrade
create-gke-cluster: build/toolchain/bin/kubectl$(EXE_EXTENSION) gcloud
	gcloud beta $(GCP_PROJECT_FLAG) container clusters create $(GKE_CLUSTER_NAME) $(GCP_LOCATION_FLAG) --cluster-version $(GKE_VERSION) --image-type cos_containerd --tags open-match $(GKE_CLUSTER_SHAPE_FLAGS) $(GKE_FUTURE_COMPAT_FLAGS)
	$(KUBECTL) create clusterrolebinding myname-cluster-admin-binding --clusterrole=cluster-admin --user=$(GCLOUD_ACCOUNT_EMAIL)

delete-gke-cluster: gcloud
	-gcloud $(GCP_PROJECT_FLAG) container clusters delete $(GKE_CLUSTER_NAME) $(GCP_LOCATION_FLAG) --quiet

create-mini-cluster: build/toolchain/bin/minikube$(EXE_EXTENSION)
	$(MINIKUBE) start --memory 6144 --cpus 4 --disk-size 50g

delete-mini-cluster: build/toolchain/bin/minikube$(EXE_EXTENSION)
	-$(MINIKUBE) delete

gcp-apply-binauthz-policy: build/policies/binauthz.yaml
ifeq ($(ENABLE_SECURITY_HARDENING),1)
	gcloud beta $(GCP_PROJECT_FLAG) container binauthz policy import build/policies/binauthz.yaml
endif

all-protos: golang-protos http-proxy-golang-protos swagger-json-docs
golang-protos: internal/pb/backend.pb.go internal/pb/frontend.pb.go internal/pb/matchfunction.pb.go internal/pb/messages.pb.go internal/pb/mmlogic.pb.go internal/pb/evaluator.pb.go

http-proxy-golang-protos: internal/pb/backend.pb.gw.go internal/pb/frontend.pb.gw.go internal/pb/matchfunction.pb.gw.go internal/pb/messages.pb.gw.go internal/pb/mmlogic.pb.gw.go internal/pb/evaluator.pb.gw.go

swagger-json-docs: api/frontend.swagger.json api/backend.swagger.json api/mmlogic.swagger.json api/matchfunction.swagger.json api/evaluator.swagger.json

internal/pb/%.pb.go: api/%.proto build/toolchain/bin/protoc$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-go$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-grpc-gateway$(EXE_EXTENSION)
	$(PROTOC) $< \
		-I $(REPOSITORY_ROOT) -I $(PROTOC_INCLUDES) \
		--go_out=plugins=grpc:$(REPOSITORY_ROOT)

internal/pb/%.pb.gw.go: api/%.proto internal/pb/%.pb.go build/toolchain/bin/protoc$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-go$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-grpc-gateway$(EXE_EXTENSION)
	$(PROTOC) $< \
		-I $(REPOSITORY_ROOT) -I $(PROTOC_INCLUDES) \
   		--grpc-gateway_out=logtostderr=true,allow_delete_body=true:$(REPOSITORY_ROOT)

api/%.swagger.json: api/%.proto internal/pb/%.pb.gw.go build/toolchain/bin/protoc$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-swagger$(EXE_EXTENSION)
	$(PROTOC) $< \
		-I $(REPOSITORY_ROOT) -I $(PROTOC_INCLUDES) --swagger_out=logtostderr=true,allow_delete_body=true:$(REPOSITORY_ROOT)

# Include structure of the protos needs to be called out do the dependency chain is run through properly.
internal/pb/backend.pb.go: internal/pb/messages.pb.go
internal/pb/frontend.pb.go: internal/pb/messages.pb.go
internal/pb/mmlogic.pb.go: internal/pb/messages.pb.go
internal/pb/evaluator.pb.go: internal/pb/messages.pb.go
internal/pb/matchfunction.pb.go: internal/pb/messages.pb.go

build:
	$(GO) build ./...

test:
	$(GO) test ./... -race -cover
	$(GO) test ./... -run -cover IgnoreRace$$
	(cd site; $(GO) test ./... -race -cover)

ci-test:
	$(GO) test ./... -race -test.count 25 -cover
	$(GO) test ./... -run IgnoreRace$$ -cover
	(cd site; $(GO) test ./... -race -test.count 25 -cover)

stress-frontend-%: build/toolchain/python/
	$(TOOLCHAIN_DIR)/python/bin/locust -f $(REPOSITORY_ROOT)/test/stress/frontend.py --host=http://localhost:51504 \
		--no-web -c $* -r 100 -t10m --csv=test/stress/stress_user$*

fmt:
	$(GO) fmt ./...
	gofmt -s -w .

vet:
	$(GO) vet ./...

golangci: build/toolchain/bin/golangci-lint$(EXE_EXTENSION)
	build/toolchain/bin/golangci-lint$(EXE_EXTENSION) run --config=.golangci.yaml

lint: fmt vet lint-chart

all: service-binaries example-binaries tools-binaries

service-binaries: cmd/minimatch/minimatch$(EXE_EXTENSION) cmd/swaggerui/swaggerui$(EXE_EXTENSION)
service-binaries: cmd/backend/backend$(EXE_EXTENSION) cmd/frontend/frontend$(EXE_EXTENSION)
service-binaries: cmd/mmlogic/mmlogic$(EXE_EXTENSION) cmd/evaluator/evaluator$(EXE_EXTENSION)

example-binaries: example-mmf-binaries
example-mmf-binaries: examples/functions/golang/soloduel/soloduel$(EXE_EXTENSION)

examples/functions/golang/soloduel/soloduel$(EXE_EXTENSION): internal/pb/mmlogic.pb.go internal/pb/mmlogic.pb.gw.go api/mmlogic.swagger.json internal/pb/matchfunction.pb.go internal/pb/matchfunction.pb.gw.go api/matchfunction.swagger.json
	cd examples/functions/golang/soloduel; $(GO_BUILD_COMMAND)

tools-binaries: tools/certgen/certgen$(EXE_EXTENSION)

cmd/backend/backend$(EXE_EXTENSION): internal/pb/backend.pb.go internal/pb/backend.pb.gw.go api/backend.swagger.json
	cd cmd/backend; $(GO_BUILD_COMMAND)

cmd/frontend/frontend$(EXE_EXTENSION): internal/pb/frontend.pb.go internal/pb/frontend.pb.gw.go api/frontend.swagger.json
	cd cmd/frontend; $(GO_BUILD_COMMAND)

cmd/mmlogic/mmlogic$(EXE_EXTENSION): internal/pb/mmlogic.pb.go internal/pb/mmlogic.pb.gw.go api/mmlogic.swagger.json
	cd cmd/mmlogic; $(GO_BUILD_COMMAND)

cmd/evaluator/evaluator$(EXE_EXTENSION): internal/pb/evaluator.pb.go internal/pb/evaluator.pb.gw.go api/evaluator.swagger.json
	cd cmd/evaluator; $(GO_BUILD_COMMAND)

# Note: This list of dependencies is long but only add file references here. If you add a .PHONY dependency make will always rebuild it.
cmd/minimatch/minimatch$(EXE_EXTENSION): internal/pb/backend.pb.go internal/pb/backend.pb.gw.go api/backend.swagger.json
cmd/minimatch/minimatch$(EXE_EXTENSION): internal/pb/frontend.pb.go internal/pb/frontend.pb.gw.go api/frontend.swagger.json
cmd/minimatch/minimatch$(EXE_EXTENSION): internal/pb/mmlogic.pb.go internal/pb/mmlogic.pb.gw.go api/mmlogic.swagger.json
cmd/minimatch/minimatch$(EXE_EXTENSION): internal/pb/evaluator.pb.go internal/pb/evaluator.pb.gw.go api/evaluator.swagger.json
cmd/minimatch/minimatch$(EXE_EXTENSION): internal/pb/matchfunction.pb.go internal/pb/matchfunction.pb.gw.go api/matchfunction.swagger.json
cmd/minimatch/minimatch$(EXE_EXTENSION): internal/pb/messages.pb.go
	cd cmd/minimatch; $(GO_BUILD_COMMAND)

cmd/swaggerui/swaggerui$(EXE_EXTENSION): site/static/swaggerui/
	cd cmd/swaggerui; $(GO_BUILD_COMMAND)

tools/certgen/certgen$(EXE_EXTENSION):
	cd tools/certgen/ && $(GO_BUILD_COMMAND)

build/policies/binauthz.yaml: install/policies/binauthz.yaml
	mkdir -p $(BUILD_DIR)/policies
	cp -f $(REPOSITORY_ROOT)/install/policies/binauthz.yaml $(BUILD_DIR)/policies/binauthz.yaml
	sed -i 's/$$PROJECT_ID/$(GCP_PROJECT_ID)/g' $(BUILD_DIR)/policies/binauthz.yaml
ifeq ($(ENABLE_SECURITY_HARDENING),1)
	sed -i 's/$$EVALUATION_MODE/ALWAYS_DENY/g' $(BUILD_DIR)/policies/binauthz.yaml
else
	sed -i 's/$$EVALUATION_MODE/ALWAYS_ALLOW/g' $(BUILD_DIR)/policies/binauthz.yaml
endif

build/certificates/: build/toolchain/bin/certgen$(EXE_EXTENSION)
	mkdir -p $(BUILD_DIR)/certificates/
	cd $(BUILD_DIR)/certificates/ && $(REPOSITORY_ROOT)/build/toolchain/bin/certgen$(EXE_EXTENSION)

node_modules/: build/toolchain/nodejs/
	-rm -r package.json package-lock.json
	-rm -rf node_modules/
	echo "{}" > package.json
	$(TOOLCHAIN_DIR)/nodejs/bin/npm install postcss-cli autoprefixer

build/site/: build/toolchain/bin/hugo$(EXE_EXTENSION) site/static/swaggerui/ node_modules/
	rm -rf build/site/
	mkdir -p build/site/
	cd site/ && ../build/toolchain/bin/hugo$(EXE_EXTENSION) --config=config.toml --source . --destination $(BUILD_DIR)/site/public/
	# Only copy the root directory since that has the AppEngine serving code.
	-cp -f site/* $(BUILD_DIR)/site
	-cp -f site/.gcloudignore $(BUILD_DIR)/site/.gcloudignore
	cp $(BUILD_DIR)/site/app.yaml $(BUILD_DIR)/site/.app.yaml

site/static/swaggerui/:
	mkdir -p $(TOOLCHAIN_DIR)/swaggerui-temp/
	mkdir -p $(TOOLCHAIN_BIN)
	curl -o $(TOOLCHAIN_DIR)/swaggerui-temp/swaggerui.zip -L \
		https://github.com/swagger-api/swagger-ui/archive/v$(SWAGGERUI_VERSION).zip
	(cd $(TOOLCHAIN_DIR)/swaggerui-temp/; unzip -q -o swaggerui.zip)
	cp -rf $(TOOLCHAIN_DIR)/swaggerui-temp/swagger-ui-$(SWAGGERUI_VERSION)/dist/ \
		$(REPOSITORY_ROOT)/site/static/swaggerui
	# Update the URL in the main page to point to a known good endpoint.
	# TODO This does not work on macOS you need to add '' after -i. This isn't build critical.
	sed -i 's/url:.*/url: \"https:\/\/open-match.dev\/api\/v0.0.0-dev\/frontend.swagger.json\",/g' $(REPOSITORY_ROOT)/site/static/swaggerui/index.html
	rm -rf $(TOOLCHAIN_DIR)/swaggerui-temp

md-test: docker
	docker run -t --rm -v $(CURDIR):/mnt:ro dkhamsing/awesome_bot --white-list "localhost,github.com/googleforgames/open-match/tree/release-,github.com/googleforgames/open-match/blob/release-,github.com/googleforgames/open-match/releases/download/v" --allow-dupe --allow-redirect --skip-save-results `find . -type f -name '*.md' -not -path './build/*' -not -path './node_modules/*' -not -path './site*' -not -path './.git*'`

site-test: TEMP_SITE_DIR := /tmp/open-match-site
site-test: build/site/ build/toolchain/bin/htmltest$(EXE_EXTENSION)
	rm -rf $(TEMP_SITE_DIR)
	mkdir -p $(TEMP_SITE_DIR)/site/
	cp -rf $(REPOSITORY_ROOT)/build/site/public/* $(TEMP_SITE_DIR)/site/
	$(HTMLTEST) --conf $(REPOSITORY_ROOT)/site/htmltest.yaml $(TEMP_SITE_DIR)

browse-site: build/site/
	cd $(BUILD_DIR)/site && dev_appserver.py .app.yaml

deploy-dev-site: build/site/ gcloud
	cd $(BUILD_DIR)/site && gcloud $(OM_SITE_GCP_PROJECT_FLAG) app deploy .app.yaml --promote --version=$(VERSION_SUFFIX) --quiet

# The website is deployed on Post Submit of every build based on the BASE_VERSION in this file.
# If the site
ci-deploy-site: build/site/ gcloud
ifeq ($(_GCB_POST_SUBMIT),1)
	@echo "Deploying website to $(GAE_SERVICE_NAME).open-match.dev version=$(GAE_SITE_VERSION)..."
	# Replace "service:"" with "service: $(GAE_SERVICE_NAME)" example, "service: 0-5"
	sed -i 's/service:.*/service: $(GAE_SERVICE_NAME)/g' $(BUILD_DIR)/site/.app.yaml
	(cd $(BUILD_DIR)/site && gcloud --quiet $(OM_SITE_GCP_PROJECT_FLAG) app deploy .app.yaml --promote --version=$(GAE_SITE_VERSION) --verbosity=info)
	# If the version matches the "latest" version from CI then also deploy to the default instance.
ifeq ($(MAJOR_MINOR_VERSION),$(_GCB_LATEST_VERSION))
	@echo "Deploying website to open-match.dev version=$(GAE_SITE_VERSION)..."
	sed -i 's/service:.*/service: default/g' $(BUILD_DIR)/site/.app.yaml
	(cd $(BUILD_DIR)/site && gcloud --quiet $(OM_SITE_GCP_PROJECT_FLAG) app deploy .app.yaml --promote --version=$(GAE_SITE_VERSION) --verbosity=info)
	# Set CORS policy on GCS bucket so that Swagger UI will work against it.
	# This only needs to be set once but in the interest of enforcing a consistency we'll apply this every deployment.
	# CORS policies signal to browsers that it's ok to use this resource in services not hosted from itself (open-match.dev)
	gsutil cors set $(REPOSITORY_ROOT)/site/gcs-cors.json gs://open-match-chart/
endif
else
	@echo "Not deploying $(GAE_SERVICE_NAME).open-match.dev because this is not a post commit change."
endif

deploy-redirect-site: gcloud
	cd $(REPOSITORY_ROOT)/site/redirect/ && gcloud $(OM_SITE_GCP_PROJECT_FLAG) app deploy app.yaml --promote --quiet

run-site: build/toolchain/bin/hugo$(EXE_EXTENSION) site/static/swaggerui/
	cd site/ && ../build/toolchain/bin/hugo$(EXE_EXTENSION) server --debug --watch --enableGitInfo . --baseURL=http://localhost:$(SITE_PORT)/ --bind 0.0.0.0 --port $(SITE_PORT) --disableFastRender

ci-deploy-artifacts: install/yaml/ swagger-json-docs gcloud
ifeq ($(_GCB_POST_SUBMIT),1)
	gsutil cp -a public-read $(REPOSITORY_ROOT)/install/yaml/* gs://open-match-chart/install/v$(BASE_VERSION)/yaml/
	gsutil cp -a public-read $(REPOSITORY_ROOT)/api/*.json gs://open-match-chart/api/v$(BASE_VERSION)/
	# TODO Add Helm Artifacts later.
	# Example: https://github.com/GoogleCloudPlatform/agones/blob/3b324a74e5e8f7049c2169ec589e627d4c8cab79/build/Makefile#L211
else
	@echo "Not deploying build artifacts to open-match.dev because this is not a post commit change."
endif

# For presubmit we want to update the protobuf generated files and verify that tests are good.
presubmit: update-deps third_party clean-protos clean-secrets all-protos lint build test clean-site site-test md-test

build/release/: presubmit clean-install-yaml install/yaml/
	mkdir -p $(BUILD_DIR)/release/
	cp $(REPOSITORY_ROOT)/install/yaml/* $(BUILD_DIR)/release/

release: REGISTRY = gcr.io/$(OPEN_MATCH_PUBLIC_IMAGES_PROJECT_ID)
release: TAG = $(BASE_VERSION)
release: build/release/

clean-secrets:
	rm -rf $(OPEN_MATCH_SECRETS_DIR)

clean-release:
	rm -rf $(REPOSITORY_ROOT)/build/release/

clean-site:
	rm -rf $(REPOSITORY_ROOT)/build/site/

clean-swagger-docs:
	rm -rf $(REPOSITORY_ROOT)/api/*.json

clean-protos:
	rm -rf $(REPOSITORY_ROOT)/internal/pb/

clean-binaries:
	rm -rf $(REPOSITORY_ROOT)/cmd/backend/backend
	rm -rf $(REPOSITORY_ROOT)/cmd/evaluator/evaluator
	rm -rf $(REPOSITORY_ROOT)/cmd/frontend/frontend
	rm -rf $(REPOSITORY_ROOT)/cmd/mmlogic/mmlogic
	rm -rf $(REPOSITORY_ROOT)/cmd/minimatch/minimatch
	rm -rf $(REPOSITORY_ROOT)/examples/functions/golang/soloduel/soloduel
	rm -rf $(REPOSITORY_ROOT)/cmd/swaggerui/swaggerui

clean-build: clean-toolchain clean-archives clean-release
	rm -rf $(REPOSITORY_ROOT)/build/

clean-toolchain:
	rm -rf $(REPOSITORY_ROOT)/build/toolchain/

clean-archives:
	rm -rf $(REPOSITORY_ROOT)/build/archives/

clean-nodejs:
	rm -rf $(REPOSITORY_ROOT)/build/toolchain/nodejs/
	rm -rf $(REPOSITORY_ROOT)/node_modules/
	rm -f $(REPOSITORY_ROOT)/package.json
	rm -f $(REPOSITORY_ROOT)/package-lock.json

clean-install-yaml:
	rm -f $(REPOSITORY_ROOT)/install/yaml/*

clean-stress-test-tools:
	rm -rf $(TOOLCHAIN_DIR)/python
	rm -f $(REPOSITORY_ROOT)/test/stress/*.csv

clean-swaggerui:
	rm -rf $(REPOSITORY_ROOT)/site/static/swaggerui/

clean: clean-images clean-binaries clean-site clean-release clean-build clean-protos clean-swagger-docs clean-nodejs clean-install-yaml clean-stress-test-tools clean-secrets clean-swaggerui

proxy-frontend: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	@echo "Frontend Health: http://localhost:$(FRONTEND_PORT)/healthz"
	@echo "Frontend RPC: http://localhost:$(FRONTEND_PORT)/debug/rpcz"
	@echo "Frontend Trace: http://localhost:$(FRONTEND_PORT)/debug/tracez"
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=open-match,component=frontend,release=$(OPEN_MATCH_CHART_NAME)" --output jsonpath='{.items[0].metadata.name}') $(FRONTEND_PORT):51504 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-backend: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	@echo "Backend Health: http://localhost:$(BACKEND_PORT)/healthz"
	@echo "Backend RPC: http://localhost:$(BACKEND_PORT)/debug/rpcz"
	@echo "Backend Trace: http://localhost:$(BACKEND_PORT)/debug/tracez"
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=open-match,component=backend,release=$(OPEN_MATCH_CHART_NAME)" --output jsonpath='{.items[0].metadata.name}') $(BACKEND_PORT):51505 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-mmlogic: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	@echo "MmLogic Health: http://localhost:$(MMLOGIC_PORT)/healthz"
	@echo "MmLogic RPC: http://localhost:$(MMLOGIC_PORT)/debug/rpcz"
	@echo "MmLogic Trace: http://localhost:$(MMLOGIC_PORT)/debug/tracez"
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=open-match,component=mmlogic,release=$(OPEN_MATCH_CHART_NAME)" --output jsonpath='{.items[0].metadata.name}') $(MMLOGIC_PORT):51503 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-evaluator: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	@echo "Evaluator Health: http://localhost:$(EVALUATOR_PORT)/healthz"
	@echo "Evaluator RPC: http://localhost:$(EVALUATOR_PORT)/debug/rpcz"
	@echo "Evaluator Trace: http://localhost:$(EVALUATOR_PORT)/debug/tracez"
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=open-match,component=evaluator,release=$(OPEN_MATCH_CHART_NAME)" --output jsonpath='{.items[0].metadata.name}') $(EVALUATOR_PORT):51506 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-grafana: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	@echo "User: admin"
	@echo "Password: openmatch"
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=grafana,release=$(OPEN_MATCH_CHART_NAME)" --output jsonpath='{.items[0].metadata.name}') $(GRAFANA_PORT):3000 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-prometheus: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=prometheus,component=server,release=$(OPEN_MATCH_CHART_NAME)" --output jsonpath='{.items[0].metadata.name}') $(PROMETHEUS_PORT):9090 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-dashboard: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	$(KUBECTL) port-forward --namespace kube-system $(shell $(KUBECTL) get pod --namespace kube-system --selector="app=kubernetes-dashboard" --output jsonpath='{.items[0].metadata.name}') $(DASHBOARD_PORT):9090 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-ui: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	@echo "SwaggerUI Health: http://localhost:$(SWAGGERUI_PORT)/"
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=open-match,component=swaggerui,release=$(OPEN_MATCH_CHART_NAME)" --output jsonpath='{.items[0].metadata.name}') $(SWAGGERUI_PORT):51500 $(PORT_FORWARD_ADDRESS_FLAG)

# Run `make proxy` instead to run everything at the same time.
# If you run this directly it will just run each proxy sequentially.
proxy-all: proxy-frontend proxy-backend proxy-mmlogic proxy-grafana proxy-prometheus proxy-evaluator proxy-ui proxy-dashboard

proxy:
	# This is an exception case where we'll call recursive make.
	# To simplify accessing all the proxy ports we'll call `make proxy-all` with enough subprocesses to run them concurrently.
	$(MAKE) proxy-all -j20

update-deps:
	$(GO) mod tidy
	cd site && $(GO) mod tidy

third_party: third_party/google/api third_party/protoc-gen-swagger/options

third_party/google/api:
	mkdir -p $(TOOLCHAIN_DIR)/googleapis-temp/
	mkdir -p $(REPOSITORY_ROOT)/third_party/google/api
	curl -o $(TOOLCHAIN_DIR)/googleapis-temp/googleapis.zip -L https://github.com/googleapis/googleapis/archive/master.zip
	(cd $(TOOLCHAIN_DIR)/googleapis-temp/; unzip -q -o googleapis.zip)
	cp -f $(TOOLCHAIN_DIR)/googleapis-temp/googleapis-master/google/api/annotations.proto \
		$(TOOLCHAIN_DIR)/googleapis-temp/googleapis-master/google/api/http.proto \
		$(TOOLCHAIN_DIR)/googleapis-temp/googleapis-master/google/api/httpbody.proto \
		$(REPOSITORY_ROOT)/third_party/google/api
	rm -rf $(TOOLCHAIN_DIR)/googleapis-temp

third_party/protoc-gen-swagger/options:
	mkdir -p $(TOOLCHAIN_DIR)/grpc-gateway-temp/
	mkdir -p $(REPOSITORY_ROOT)/third_party/protoc-gen-swagger/options
	curl -o $(TOOLCHAIN_DIR)/grpc-gateway-temp/grpc-gateway.zip -L https://github.com/grpc-ecosystem/grpc-gateway/archive/master.zip
	(cd $(TOOLCHAIN_DIR)/grpc-gateway-temp/; unzip -q -o grpc-gateway.zip)
	cp -f $(TOOLCHAIN_DIR)/grpc-gateway-temp/grpc-gateway-master/protoc-gen-swagger/options/annotations.proto \
		$(TOOLCHAIN_DIR)/grpc-gateway-temp/grpc-gateway-master/protoc-gen-swagger/options/openapiv2.proto \
		$(REPOSITORY_ROOT)/third_party/protoc-gen-swagger/options
	rm -rf $(TOOLCHAIN_DIR)/grpc-gateway-temp

sync-deps:
	$(GO) mod download
	cd site && $(GO) mod download

sleep-10:
	sleep 10

# Prevents users from running with sudo.
# There's an exception for Google Cloud Build because it runs as root.
no-sudo:
ifndef ALLOW_BUILD_WITH_SUDO
ifeq ($(shell whoami),root)
	@echo "ERROR: Running Makefile as root (or sudo)"
	@echo "Please follow the instructions at https://docs.docker.com/install/linux/linux-postinstall/ if you are trying to sudo run the Makefile because of the 'Cannot connect to the Docker daemon' error."
	@echo "NOTE: sudo/root do not have the authentication token to talk to any GCP service via gcloud."
	exit 1
endif
endif

.PHONY: docker gcloud deploy-redirect-site update-deps sync-deps sleep-10 proxy-dashboard proxy-prometheus proxy-grafana clean clean-build clean-toolchain clean-archives clean-binaries clean-protos presubmit test ci-test site-test md-test vet

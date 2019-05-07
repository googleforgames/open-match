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

## NOTICE: There's 2 variables you need to make sure are set.
## GCP_PROJECT_ID if you're working against GCP.
## Or $REGISTRY if you want to use your own custom docker registry.
##
## Basic Deployment
## make create-gke-cluster OR make create-mini-cluster
## make push-helm
## make REGISTRY=gcr.io/$PROJECT_ID push-images -j$(nproc)
## make install-chart
## Generate Files
## make all-protos
##
## Building
## make all -j$(nproc)
##
## Access monitoring
## make proxy-prometheus
## make proxy-grafana
##
## Run those tools
## make run-backendclient
## make run-frontendclient
## make run-clientloadgen
##
## Teardown
## make delete-mini-cluster
## make delete-gke-cluster
##
# http://makefiletutorial.com/

BASE_VERSION = 0.0.0-dev
VERSION_SUFFIX = $(shell git rev-parse --short=7 HEAD | tr -d [:punct:])
BRANCH_NAME = $(shell git rev-parse --abbrev-ref HEAD | tr -d [:punct:])
VERSION = $(BASE_VERSION)-$(VERSION_SUFFIX)

PROTOC_VERSION = 3.7.1
HELM_VERSION = 2.13.1
HUGO_VERSION = 0.55.4
KUBECTL_VERSION = 1.14.1
NODEJS_VERSION = 10.15.3
SKAFFOLD_VERSION = latest
MINIKUBE_VERSION = latest
HTMLTEST_VERSION = 0.10.1
GOLANGCI_VERSION = 1.16.0

PROTOC_RELEASE_BASE = https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/protoc-$(PROTOC_VERSION)
GO = GO111MODULE=on go
# Defines the absolute local directory of the open-match project
REPOSITORY_ROOT := $(patsubst %/,%,$(dir $(abspath $(MAKEFILE_LIST))))
GO_BUILD_COMMAND = CGO_ENABLED=0 $(GO) build -a -installsuffix cgo .
BUILD_DIR = $(REPOSITORY_ROOT)/build
TOOLCHAIN_DIR = $(BUILD_DIR)/toolchain
TOOLCHAIN_BIN = $(TOOLCHAIN_DIR)/bin
PROTOC := $(TOOLCHAIN_BIN)/protoc
PROTOC_INCLUDES := $(TOOLCHAIN_DIR)/include/
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
LOCAL_CLOUD_BUILD_PUSH = # --push
KUBECTL_RUN_ENV = --env='REDIS_SERVICE_HOST=$$(OM_REDIS_MASTER_SERVICE_HOST)' --env='REDIS_SERVICE_PORT=$$(OM_REDIS_MASTER_SERVICE_PORT)'
GCP_LOCATION_FLAG = --zone $(GCP_ZONE)
# Flags to simulate behavior of newer versions of Kubernetes
KUBERNETES_COMPAT = --no-enable-basic-auth --no-issue-client-certificate --enable-ip-alias --metadata disable-legacy-endpoints=true --enable-autoupgrade
GO111MODULE = on
PROMETHEUS_PORT = 9090
GRAFANA_PORT = 3000
SITE_PORT = 8080
HELM = $(TOOLCHAIN_BIN)/helm
TILLER = $(TOOLCHAIN_BIN)/tiller
MINIKUBE = $(TOOLCHAIN_BIN)/minikube
KUBECTL = $(TOOLCHAIN_BIN)/kubectl
HTMLTEST = $(TOOLCHAIN_BIN)/htmltest
SERVICE = default
OPEN_MATCH_CHART_NAME = open-match
OPEN_MATCH_KUBERNETES_NAMESPACE = open-match
OPEN_MATCH_EXAMPLE_CHART_NAME = open-match-example
OPEN_MATCH_EXAMPLE_KUBERNETES_NAMESPACE = open-match
REDIS_NAME = om-redis
GCLOUD_ACCOUNT_EMAIL = $(shell gcloud auth list --format yaml | grep account: | cut -c 10-)
_GCB_POST_SUBMIT ?= 0
DEV_SITE_VERSION = head

# Make port forwards accessible outside of the proxy machine.
PORT_FORWARD_ADDRESS_FLAG = --address 0.0.0.0
DASHBOARD_PORT = 9092
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
	PROTOC_PACKAGE = $(PROTOC_RELEASE_BASE)-win64.zip
	KUBECTL_PACKAGE = https://storage.googleapis.com/kubernetes-release/release/v$(KUBECTL_VERSION)/bin/windows/amd64/kubectl.exe
	HUGO_PACKAGE = https://github.com/gohugoio/hugo/releases/download/v$(HUGO_VERSION)/hugo_extended_$(HUGO_VERSION)_Windows-64bit.zip
	NODEJS_PACKAGE = https://nodejs.org/dist/v$(NODEJS_VERSION)/node-v$(NODEJS_VERSION)-win-x64.zip
	NODEJS_PACKAGE_NAME = nodejs.zip
	HTMLTEST_PACKAGE = https://github.com/wjdp/htmltest/releases/download/v$(HTMLTEST_VERSION)/htmltest_$(HTMLTEST_VERSION)_windows_amd64.zip
	GOLANGCI_PACKAGE = https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_VERSION)/golangci-lint-$(GOLANGCI_VERSION)-windows-amd64.zip
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		HELM_PACKAGE = https://storage.googleapis.com/kubernetes-helm/helm-v$(HELM_VERSION)-linux-amd64.tar.gz
		MINIKUBE_PACKAGE = https://storage.googleapis.com/minikube/releases/$(MINIKUBE_VERSION)/minikube-linux-amd64
		SKAFFOLD_PACKAGE = https://storage.googleapis.com/skaffold/releases/$(SKAFFOLD_VERSION)/skaffold-linux-amd64
		PROTOC_PACKAGE = $(PROTOC_RELEASE_BASE)-linux-x86_64.zip
		KUBECTL_PACKAGE = https://storage.googleapis.com/kubernetes-release/release/v$(KUBECTL_VERSION)/bin/linux/amd64/kubectl
		HUGO_PACKAGE = https://github.com/gohugoio/hugo/releases/download/v$(HUGO_VERSION)/hugo_extended_$(HUGO_VERSION)_Linux-64bit.tar.gz
		NODEJS_PACKAGE = https://nodejs.org/dist/v$(NODEJS_VERSION)/node-v$(NODEJS_VERSION)-linux-x64.tar.gz
		NODEJS_PACKAGE_NAME = nodejs.tar.gz
		HTMLTEST_PACKAGE = https://github.com/wjdp/htmltest/releases/download/v$(HTMLTEST_VERSION)/htmltest_$(HTMLTEST_VERSION)_linux_amd64.tar.gz
		GOLANGCI_PACKAGE = https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_VERSION)/golangci-lint-$(GOLANGCI_VERSION)-linux-amd64.tar.gz
	endif
	ifeq ($(UNAME_S),Darwin)
		HELM_PACKAGE = https://storage.googleapis.com/kubernetes-helm/helm-v$(HELM_VERSION)-darwin-amd64.tar.gz
		MINIKUBE_PACKAGE = https://storage.googleapis.com/minikube/releases/$(MINIKUBE_VERSION)/minikube-darwin-amd64
		SKAFFOLD_PACKAGE = https://storage.googleapis.com/skaffold/releases/$(SKAFFOLD_VERSION)/skaffold-darwin-amd64
		PROTOC_PACKAGE = $(PROTOC_RELEASE_BASE)-osx-x86_64.zip
		KUBECTL_PACKAGE = https://storage.googleapis.com/kubernetes-release/release/v$(KUBECTL_VERSION)/bin/darwin/amd64/kubectl
		HUGO_PACKAGE = https://github.com/gohugoio/hugo/releases/download/v$(HUGO_VERSION)/hugo_extended_$(HUGO_VERSION)_macOS-64bit.tar.gz
		NODEJS_PACKAGE = https://nodejs.org/dist/v$(NODEJS_VERSION)/node-v$(NODEJS_VERSION)-darwin-x64.tar.gz
		NODEJS_PACKAGE_NAME = nodejs.tar.gz
		HTMLTEST_PACKAGE = https://github.com/wjdp/htmltest/releases/download/v$(HTMLTEST_VERSION)/htmltest_$(HTMLTEST_VERSION)_osx_amd64.tar.gz
		GOLANGCI_PACKAGE = https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_VERSION)/golangci-lint-$(GOLANGCI_VERSION)-darwin-amd64.tar.gz
	endif
endif

help:
	@cat Makefile | grep ^\#\# | grep -v ^\#\#\# |cut -c 4-

local-cloud-build: gcloud
	cloud-build-local --config=cloudbuild.yaml --dryrun=false $(LOCAL_CLOUD_BUILD_PUSH) --substitutions SHORT_SHA=$(VERSION_SUFFIX),_GCB_POST_SUBMIT=$(_GCB_POST_SUBMIT),BRANCH_NAME=$(BRANCH_NAME) .

push-images: push-service-images deprecated-push-images
push-service-images: push-backend-image push-frontend-image  push-mmlogic-image push-minimatch-image

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

build-images: build-service-images deprecated-build-images
build-service-images: build-backend-image build-frontend-image build-mmlogic-image build-minimatch-image

build-base-build-image: docker
	docker build -f Dockerfile.base-build -t open-match-base-build .

build-backend-image: docker build-base-build-image
	docker build -f cmd/future/backend/Dockerfile -t $(REGISTRY)/openmatch-backend:$(TAG) -t $(REGISTRY)/openmatch-backend:$(ALTERNATE_TAG) .

build-frontend-image: docker build-base-build-image
	docker build -f cmd/future/frontend/Dockerfile -t $(REGISTRY)/openmatch-frontend:$(TAG) -t $(REGISTRY)/openmatch-frontend:$(ALTERNATE_TAG) .

build-mmlogic-image: docker build-base-build-image
	docker build -f cmd/future/mmlogic/Dockerfile -t $(REGISTRY)/openmatch-mmlogic:$(TAG) -t $(REGISTRY)/openmatch-mmlogic:$(ALTERNATE_TAG) .

build-minimatch-image: docker build-base-build-image
	docker build -f cmd/future/minimatch/Dockerfile -t $(REGISTRY)/openmatch-minimatch:$(TAG) -t $(REGISTRY)/openmatch-minimatch:$(ALTERNATE_TAG) .

clean-images: docker deprecated-clean-images
	-docker rmi -f open-match-base-build

	-docker rmi -f $(REGISTRY)/openmatch-backend:$(TAG) $(REGISTRY)/openmatch-backend:$(ALTERNATE_TAG)
	-docker rmi -f $(REGISTRY)/openmatch-frontend:$(TAG) $(REGISTRY)/openmatch-frontend:$(ALTERNATE_TAG)
	-docker rmi -f $(REGISTRY)/openmatch-mmlogic:$(TAG) $(REGISTRY)/openmatch-mmlogic:$(ALTERNATE_TAG)
	-docker rmi -f $(REGISTRY)/openmatch-minimatch:$(TAG) $(REGISTRY)/openmatch-minimatch:$(ALTERNATE_TAG)

install-redis: build/toolchain/bin/helm$(EXE_EXTENSION)
	$(HELM) upgrade --install --wait --debug $(REDIS_NAME) stable/redis --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE)

update-chart-deps: build/toolchain/bin/helm$(EXE_EXTENSION)
	(cd install/helm/open-match; $(HELM) repo add incubator https://kubernetes-charts-incubator.storage.googleapis.com; $(HELM) dependency update)

lint-chart: build/toolchain/bin/helm$(EXE_EXTENSION)
	(cd install/helm; $(HELM) lint open-match; $(HELM) lint open-match-example)

print-chart: build/toolchain/bin/helm$(EXE_EXTENSION)
	(cd install/helm; $(HELM) install --dry-run --debug open-match; $(HELM) install --dry-run --debug open-match-example)

install-chart: build/toolchain/bin/helm$(EXE_EXTENSION)
	$(HELM) upgrade --install --wait --debug $(OPEN_MATCH_CHART_NAME) install/helm/open-match \
		--timeout=400 \
		--namespace=$(OPEN_MATCH_KUBERNETES_NAMESPACE) \
		--set openmatch.image.registry=$(REGISTRY) \
		--set openmatch.image.tag=$(TAG)

install-example-chart: build/toolchain/bin/helm$(EXE_EXTENSION)
	$(HELM) upgrade --install --wait --debug $(OPEN_MATCH_EXAMPLE_CHART_NAME) install/helm/open-match-example \
	  --namespace=$(OPEN_MATCH_KUBERNETES_NAMESPACE) \
	  --set openmatch.image.registry=$(REGISTRY) \
	  --set openmatch.image.tag=$(TAG)

delete-example-chart: build/toolchain/bin/helm$(EXE_EXTENSION)
	-$(HELM) delete --purge $(OPEN_MATCH_EXAMPLE_CHART_NAME)

dry-chart: build/toolchain/bin/helm$(EXE_EXTENSION)
	$(HELM) upgrade --install --wait --debug --dry-run $(OPEN_MATCH_CHART_NAME) install/helm/open-match \
		--namespace=$(OPEN_MATCH_KUBERNETES_NAMESPACE) \
		--set openmatch.image.registry=$(REGISTRY) \
		--set openmatch.image.tag=$(TAG)

delete-chart: build/toolchain/bin/helm$(EXE_EXTENSION) build/toolchain/bin/kubectl$(EXE_EXTENSION)
	-$(HELM) delete --purge $(OPEN_MATCH_CHART_NAME)
	-$(KUBECTL) delete crd prometheuses.monitoring.coreos.com
	-$(KUBECTL) delete crd servicemonitors.monitoring.coreos.com
	-$(KUBECTL) delete crd prometheusrules.monitoring.coreos.com

update-helm-deps: build/toolchain/bin/helm$(EXE_EXTENSION)
	(cd install/helm/open-match; $(HELM) dependencies update)

install/yaml/: install/yaml/install.yaml install/yaml/install-example.yaml install/yaml/01-redis-chart.yaml install/yaml/02-open-match.yaml install/yaml/03-prometheus-chart.yaml install/yaml/04-grafana-chart.yaml

install/yaml/01-redis-chart.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template --name $(OPEN_MATCH_CHART_NAME) --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) \
		--set redis.fullnameOverride='$(REDIS_NAME)' \
		--set openmatch.config.install=false \
		--set openmatch.backendapi.install=false \
		--set openmatch.frontendapi.install=false \
		--set openmatch.mmlogicapi.install=false \
		--set prometheus.enabled=false \
		--set grafana.enabled=false \
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
		--set openmatch.backendapi.install=false \
		--set openmatch.frontendapi.install=false \
		--set openmatch.mmlogicapi.install=false \
		--set grafana.enabled=false \
		install/helm/open-match > install/yaml/03-prometheus-chart.yaml

install/yaml/04-grafana-chart.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template --name $(OPEN_MATCH_CHART_NAME) --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) \
		--set redis.enabled=false \
		--set openmatch.config.install=false \
		--set openmatch.backendapi.install=false \
		--set openmatch.frontendapi.install=false \
		--set openmatch.mmlogicapi.install=false \
		--set prometheus.enabled=false \
		--set grafana.enabled=true \
		install/helm/open-match > install/yaml/04-grafana-chart.yaml

install/yaml/install.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template --name $(OPEN_MATCH_CHART_NAME) --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) \
		--set openmatch.image.registry=$(REGISTRY) \
		--set openmatch.image.tag=$(TAG) \
		--set redis.enabled=true \
		--set prometheus.enabled=true \
		--set grafana.enabled=true \
		install/helm/open-match > install/yaml/install.yaml

install/yaml/install-example.yaml: build/toolchain/bin/helm$(EXE_EXTENSION)
	mkdir -p install/yaml/
	$(HELM) template --name $(OPEN_MATCH_EXAMPLE_CHART_NAME) --namespace $(OPEN_MATCH_EXAMPLE_KUBERNETES_NAMESPACE) \
		--set openmatch.image.registry=$(REGISTRY) \
		--set openmatch.image.tag=$(TAG) \
		install/helm/open-match-example > install/yaml/install-example.yaml

set-redis-password:
	@stty -echo; \
		printf "Redis password: "; \
		read REDIS_PASSWORD; \
		stty echo; \
		printf "\n"; \
		$(KUBECTL) create secret generic $(REDIS_NAME) -n $(OPEN_MATCH_EXAMPLE_KUBERNETES_NAMESPACE) --from-literal=redis-password=$$REDIS_PASSWORD --dry-run -o yaml | $(KUBECTL) replace -f - --force

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

build/toolchain/python/:
	virtualenv --python=python3 $(TOOLCHAIN_DIR)/python/
	# Hack to workaround some crazy bug in pip that's chopping off python executable's name.
	cd build/toolchain/python/bin && ln -s python3 pytho
	cd build/toolchain/python/ && . bin/activate && pip install locustio && deactivate

build/toolchain/include/google/api/:
	mkdir -p $(TOOLCHAIN_DIR)/googleapis-temp/
	mkdir -p $(TOOLCHAIN_BIN)
	curl -o $(TOOLCHAIN_DIR)/googleapis-temp/googleapis.zip -L \
		https://github.com/googleapis/googleapis/archive/master.zip
	(cd $(TOOLCHAIN_DIR)/googleapis-temp/; unzip -q -o googleapis.zip)
	cp -rf $(TOOLCHAIN_DIR)/googleapis-temp/googleapis-master/google/api/ \
		$(PROTOC_INCLUDES)/google/api
	rm -rf $(TOOLCHAIN_DIR)/googleapis-temp

build/toolchain/include/protoc-gen-swagger/:
	mkdir -p $(TOOLCHAIN_DIR)/grpc-gateway-temp/
	mkdir -p $(TOOLCHAIN_BIN)
	mkdir -p $(PROTOC_INCLUDES)/protoc-gen-swagger/options/
	curl -o $(TOOLCHAIN_DIR)/grpc-gateway-temp/grpc-gateway.zip -L \
		https://github.com/grpc-ecosystem/grpc-gateway/archive/master.zip
	(cd $(TOOLCHAIN_DIR)/grpc-gateway-temp/; unzip -q -o grpc-gateway.zip)
	cp -rf $(TOOLCHAIN_DIR)/grpc-gateway-temp/grpc-gateway-master/protoc-gen-swagger/options/*.proto \
		$(PROTOC_INCLUDES)/protoc-gen-swagger/options/
	rm -rf $(TOOLCHAIN_DIR)/grpc-gateway-temp

build/toolchain/bin/protoc$(EXE_EXTENSION):
	mkdir -p $(TOOLCHAIN_BIN)
	curl -o $(TOOLCHAIN_DIR)/protoc-temp.zip -L $(PROTOC_PACKAGE)
	(cd $(TOOLCHAIN_DIR); unzip -q -o protoc-temp.zip)
	rm $(TOOLCHAIN_DIR)/protoc-temp.zip $(TOOLCHAIN_DIR)/readme.txt

build/toolchain/bin/protoc-gen-go$(EXE_EXTENSION): build/toolchain/include/google/api/ build/toolchain/include/protoc-gen-swagger/
	mkdir -p $(TOOLCHAIN_BIN)
	cd $(TOOLCHAIN_BIN) && $(GO) build -pkgdir . github.com/golang/protobuf/protoc-gen-go

build/toolchain/bin/protoc-gen-grpc-gateway$(EXE_EXTENSION): build/toolchain/include/google/api/ build/toolchain/include/protoc-gen-swagger/
	cd $(TOOLCHAIN_BIN) && $(GO) build -pkgdir . github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway

build/toolchain/bin/protoc-gen-swagger$(EXE_EXTENSION): build/toolchain/include/google/api/ build/toolchain/include/protoc-gen-swagger/
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
	echo "Waiting for Tiller to become ready..."
	$(KUBECTL) wait deployment --timeout=60s --for condition=available -l app=helm,name=tiller --namespace kube-system

delete-helm: build/toolchain/bin/helm$(EXE_EXTENSION) build/toolchain/bin/kubectl$(EXE_EXTENSION)
	-$(HELM) reset
	-$(KUBECTL) delete serviceaccount --namespace kube-system tiller
	-$(KUBECTL) delete clusterrolebinding tiller-cluster-rule
ifneq ($(strip $($(KUBECTL) get clusterroles | grep -i rbac)),)
	-$(KUBECTL) delete deployment --namespace kube-system tiller-deploy
endif
	echo "Waiting for Tiller to go away..."
	-$(KUBECTL) wait deployment --timeout=60s --for delete -l app=helm,name=tiller --namespace kube-system

# Fake target for docker
docker: no-sudo

# Fake target for gcloud
gcloud: no-sudo

auth-docker: gcloud docker
	gcloud $(GCP_PROJECT_FLAG) auth configure-docker

auth-gke-cluster: gcloud
	gcloud $(GCP_PROJECT_FLAG) container clusters get-credentials $(GKE_CLUSTER_NAME) $(GCP_LOCATION_FLAG)

create-gke-cluster: build/toolchain/bin/kubectl$(EXE_EXTENSION) gcloud
	gcloud $(GCP_PROJECT_FLAG) container clusters create $(GKE_CLUSTER_NAME) $(GCP_LOCATION_FLAG) --machine-type n1-standard-4 --tags open-match $(KUBERNETES_COMPAT)
	$(KUBECTL) create clusterrolebinding myname-cluster-admin-binding --clusterrole=cluster-admin --user=$(GCLOUD_ACCOUNT_EMAIL)

delete-gke-cluster: gcloud
	gcloud $(GCP_PROJECT_FLAG) container clusters delete $(GKE_CLUSTER_NAME) $(GCP_LOCATION_FLAG) --quiet

create-mini-cluster: build/toolchain/bin/minikube$(EXE_EXTENSION)
	$(MINIKUBE) start --memory 6144 --cpus 4 --disk-size 50g

delete-mini-cluster: build/toolchain/bin/minikube$(EXE_EXTENSION)
	$(MINIKUBE) delete

all-protos: deprecated-all-protos golang-protos http-proxy-golang-protos swagger-json-docs
golang-protos: internal/future/pb/backend.pb.go internal/future/pb/frontend.pb.go internal/future/pb/matchfunction.pb.go internal/future/pb/messages.pb.go internal/future/pb/mmlogic.pb.go

http-proxy-golang-protos: internal/future/pb/backend.pb.gw.go internal/future/pb/frontend.pb.gw.go internal/future/pb/matchfunction.pb.gw.go internal/future/pb/messages.pb.gw.go internal/future/pb/mmlogic.pb.gw.go

swagger-json-docs: api/frontend.swagger.json api/backend.swagger.json api/mmlogic.swagger.json api/matchfunction.swagger.json

internal/future/pb/%.pb.go: api/%.proto build/toolchain/bin/protoc$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-go$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-grpc-gateway$(EXE_EXTENSION)
	$(PROTOC) $< \
		-I $(REPOSITORY_ROOT) -I $(PROTOC_INCLUDES) \
		--go_out=plugins=grpc:$(REPOSITORY_ROOT)

internal/future/pb/%.pb.gw.go: api/%.proto internal/future/pb/%.pb.go build/toolchain/bin/protoc$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-go$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-grpc-gateway$(EXE_EXTENSION)
	$(PROTOC) $< \
		-I $(REPOSITORY_ROOT) -I $(PROTOC_INCLUDES) \
   		--grpc-gateway_out=logtostderr=true,allow_delete_body=true:$(REPOSITORY_ROOT)

api/%.swagger.json: api/%.proto internal/future/pb/%.pb.gw.go build/toolchain/bin/protoc$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-swagger$(EXE_EXTENSION)
	$(PROTOC) $< \
		-I $(REPOSITORY_ROOT) -I $(PROTOC_INCLUDES) --swagger_out=logtostderr=true,allow_delete_body=true:$(REPOSITORY_ROOT)

# Include structure of the protos needs to be called out do the dependency chain is run through properly.
internal/future/pb/backend.pb.go: internal/future/pb/messages.pb.go
internal/future/pb/frontend.pb.go: internal/future/pb/messages.pb.go
internal/future/pb/mmlogic.pb.go: internal/future/pb/messages.pb.go
internal/future/pb/matchfunction.pb.go: internal/future/pb/messages.pb.go

build:
	$(GO) build ./...

test:
	$(GO) test ./... -race -cover
	$(GO) test ./... -run -cover IgnoreRace$$

ci-test:
	$(GO) test ./... -race -test.count 25 -cover
	$(GO) test ./... -run IgnoreRace$$ -cover

stress-test-%: build/toolchain/python/
	$(TOOLCHAIN_DIR)/python/bin/locust -f $(REPOSITORY_ROOT)/test/stress/frontend.py --host=http://localhost:51504 \
		--no-web -c $* -r 100 -t10m --csv=test/stress/stress_user$*

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

# Blocked on https://github.com/golangci/golangci-lint/issues/500 to be added to `make lint`
golangci: build/toolchain/bin/golangci-lint$(EXE_EXTENSION)
	build/toolchain/bin/golangci-lint$(EXE_EXTENSION) run -v --config=.golangci.yaml

lint: fmt vet lint-chart

all: service-binaries deprecated-all
service-binaries: cmd/future/minimatch/minimatch$(EXE_EXTENSION) cmd/future/backend/backend$(EXE_EXTENSION) cmd/future/frontend/frontend$(EXE_EXTENSION) cmd/future/mmlogic/mmlogic$(EXE_EXTENSION)
tools-binaries: tools/certgen/certgen$(EXE_EXTENSION)

cmd/future/backend/backend$(EXE_EXTENSION): internal/future/pb/backend.pb.go internal/future/pb/backend.pb.gw.go api/backend.swagger.json
	cd cmd/future/backend; $(GO_BUILD_COMMAND)

cmd/future/frontend/frontend$(EXE_EXTENSION): internal/future/pb/frontend.pb.go internal/future/pb/frontend.pb.gw.go api/frontend.swagger.json
	cd cmd/future/frontend; $(GO_BUILD_COMMAND)

cmd/future/mmlogic/mmlogic$(EXE_EXTENSION): internal/future/pb/mmlogic.pb.go internal/future/pb/mmlogic.pb.gw.go api/mmlogic.swagger.json
	cd cmd/future/mmlogic; $(GO_BUILD_COMMAND)

# Note: This list of dependencies is long but only add file references here. If you add a .PHONY dependency make will always rebuild it.
cmd/future/minimatch/minimatch$(EXE_EXTENSION): internal/future/pb/backend.pb.go internal/future/pb/backend.pb.gw.go api/backend.swagger.json
cmd/future/minimatch/minimatch$(EXE_EXTENSION): internal/future/pb/frontend.pb.go internal/future/pb/frontend.pb.gw.go api/frontend.swagger.json
cmd/future/minimatch/minimatch$(EXE_EXTENSION): internal/future/pb/mmlogic.pb.go internal/future/pb/mmlogic.pb.gw.go api/mmlogic.swagger.json
cmd/future/minimatch/minimatch$(EXE_EXTENSION): internal/future/pb/messages.pb.go
	cd cmd/future/minimatch; $(GO_BUILD_COMMAND)

tools/certgen/certgen$(EXE_EXTENSION):
	cd tools/certgen/ && $(GO_BUILD_COMMAND)

build/certificates/: build/toolchain/bin/certgen$(EXE_EXTENSION)
	mkdir -p $(BUILD_DIR)/certificates/
	cd $(BUILD_DIR)/certificates/ && $(REPOSITORY_ROOT)/build/toolchain/bin/certgen$(EXE_EXTENSION)

node_modules/: build/toolchain/nodejs/
	-rm -r package.json package-lock.json
	-rm -rf node_modules/
	echo "{}" > package.json
	$(TOOLCHAIN_DIR)/nodejs/bin/npm install postcss-cli autoprefixer

build/site/: build/toolchain/bin/hugo$(EXE_EXTENSION) node_modules/
	rm -rf build/site/
	mkdir -p build/site/
	cd site/ && ../build/toolchain/bin/hugo$(EXE_EXTENSION) --config=config.toml --source . --destination $(BUILD_DIR)/site/public/
	# Only copy the root directory since that has the AppEngine serving code.
	-cp -f site/* $(BUILD_DIR)/site
	-cp -f site/.gcloudignore $(BUILD_DIR)/site/.gcloudignore
	#cd $(BUILD_DIR)/site && "SERVICE=$(SERVICE) envsubst < app.yaml > .app.yaml"
	cp $(BUILD_DIR)/site/app.yaml $(BUILD_DIR)/site/.app.yaml

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

ci-deploy-dev-site: build/site/ gcloud
ifeq ($(_GCB_POST_SUBMIT),1)
	echo "Deploying website to development.open-match.dev..."
	cd $(BUILD_DIR)/site && find .
	cd $(BUILD_DIR)/site && pwd && gcloud $(OM_SITE_GCP_PROJECT_FLAG) app deploy .app.yaml --promote --version=$(DEV_SITE_VERSION) --verbosity=info
else
	echo "Not deploying development.open-match.dev because this is not a post commit change."
endif

deploy-redirect-site: gcloud
	cd $(REPOSITORY_ROOT)/site/redirect/ && gcloud $(OM_SITE_GCP_PROJECT_FLAG) app deploy app.yaml --promote --quiet

run-site: build/toolchain/bin/hugo$(EXE_EXTENSION)
	cd site/ && ../build/toolchain/bin/hugo$(EXE_EXTENSION) server --debug --watch --enableGitInfo . --baseURL=http://localhost:$(SITE_PORT)/ --bind 0.0.0.0 --port $(SITE_PORT) --disableFastRender

ci-deploy-artifacts: install/yaml/ gcloud
ifeq ($(_GCB_POST_SUBMIT),1)
	#gsutil cp -a public-read $(REPOSITORY_ROOT)/install/yaml/* gs://open-match-chart/install/$(VERSION_SUFFIX)/
	gsutil cp -a public-read $(REPOSITORY_ROOT)/install/yaml/* gs://open-match-chart/install/yaml/$(BRANCH_NAME)-latest/
else
	echo "Not deploying development.open-match.dev because this is not a post commit change."
endif

# For presubmit we want to update the protobuf generated files and verify that tests are good.
presubmit: update-deps clean-protos all-protos lint build test clean-site site-test

build/release/: presubmit clean-install-yaml install/yaml/
	mkdir -p $(BUILD_DIR)/release/
	cp $(REPOSITORY_ROOT)/install/yaml/* $(BUILD_DIR)/release/

release: REGISTRY = gcr.io/$(OPEN_MATCH_PUBLIC_IMAGES_PROJECT_ID)
release: TAG = $(BASE_VERSION)
release: build/release/

clean-release:
	rm -rf $(REPOSITORY_ROOT)/build/release/

clean-site:
	rm -rf $(REPOSITORY_ROOT)/build/site/

clean-swagger-docs: deprecated-clean-swagger-docs
	rm -rf $(REPOSITORY_ROOT)/api/*.json

clean-protos: deprecated-clean-protos
	rm -rf $(REPOSITORY_ROOT)/internal/future/pb/

clean-binaries: deprecated-clean-binaries
	rm -rf $(REPOSITORY_ROOT)/cmd/future/backend/backend
	rm -rf $(REPOSITORY_ROOT)/cmd/future/frontend/frontend
	rm -rf $(REPOSITORY_ROOT)/cmd/future/mmlogic/mmlogic
	rm -rf $(REPOSITORY_ROOT)/cmd/future/minimatch/minimatch

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
	rm -f $(REPOSITORY_ROOT)/install/yaml/install.yaml
	rm -f $(REPOSITORY_ROOT)/install/yaml/install-example.yaml
	rm -f $(REPOSITORY_ROOT)/install/yaml/01-redis-chart.yaml
	rm -f $(REPOSITORY_ROOT)/install/yaml/02-open-match.yaml
	rm -f $(REPOSITORY_ROOT)/install/yaml/03-prometheus-chart.yaml
	rm -f $(REPOSITORY_ROOT)/install/yaml/04-grafana-chart.yaml

clean-stress-test-tools:
	rm -rf $(TOOLCHAIN_DIR)/python
	rm -f $(REPOSITORY_ROOT)/test/stress/*.csv

clean: clean-images clean-binaries clean-site clean-release clean-build clean-protos clean-swagger-docs clean-nodejs clean-install-yaml clean-stress-test-tools

proxy-grafana: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	echo "User: admin"
	echo "Password: openmatch"
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=grafana,release=$(OPEN_MATCH_CHART_NAME)" --output jsonpath='{.items[0].metadata.name}') $(GRAFANA_PORT):3000 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-prometheus: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=prometheus,component=server,release=$(OPEN_MATCH_CHART_NAME)" --output jsonpath='{.items[0].metadata.name}') $(PROMETHEUS_PORT):9090 $(PORT_FORWARD_ADDRESS_FLAG)

proxy-dashboard: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	$(KUBECTL) port-forward --namespace kube-system $(shell $(KUBECTL) get pod --namespace kube-system --selector="app=kubernetes-dashboard" --output jsonpath='{.items[0].metadata.name}') $(DASHBOARD_PORT):9090 $(PORT_FORWARD_ADDRESS_FLAG)

update-deps:
	$(GO) mod tidy
	cd site && $(GO) mod tidy

proxy-frontend: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	$(KUBECTL) port-forward --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) $(shell $(KUBECTL) get pod --namespace $(OPEN_MATCH_KUBERNETES_NAMESPACE) --selector="app=open-match,component=frontend,release=$(OPEN_MATCH_CHART_NAME)" --output jsonpath='{.items[0].metadata.name}') 51504:51504 $(PORT_FORWARD_ADDRESS_FLAG) 

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
	echo "ERROR: Running Makefile as root (or sudo)"
	echo "Please follow the instructions at https://docs.docker.com/install/linux/linux-postinstall/ if you are trying to sudo run the Makefile because of the 'Cannot connect to the Docker daemon' error."
	echo "NOTE: sudo/root do not have the authentication token to talk to any GCP service via gcloud."
	exit 1
endif
endif

.PHONY: docker gcloud deploy-redirect-site update-deps sync-deps sleep-10 proxy-dashboard proxy-prometheus proxy-grafana clean clean-build clean-toolchain clean-archives clean-binaries clean-protos presubmit test ci-test vet

################################################################################
#                              Deprecated Targets                              #
################################################################################

deprecated-push-images: deprecated-push-service-images deprecated-push-client-images deprecated-push-mmf-example-images deprecated-push-evaluator-example-images
deprecated-push-service-images: deprecated-push-frontendapi-image deprecated-push-backendapi-image deprecated-push-mmlogicapi-image
deprecated-push-mmf-example-images: deprecated-push-mmf-go-grpc-serving-simple-image
deprecated-push-client-images: deprecated-push-backendclient-image deprecated-push-clientloadgen-image deprecated-push-frontendclient-image
deprecated-push-evaluator-example-images: deprecated-push-evaluator-serving-image

# Deprecated
deprecated-push-frontendapi-image: docker deprecated-build-frontendapi-image
	docker push $(REGISTRY)/openmatch-frontendapi:$(TAG)
	docker push $(REGISTRY)/openmatch-frontendapi:$(ALTERNATE_TAG)

# Deprecated
deprecated-push-backendapi-image: docker deprecated-build-backendapi-image
	docker push $(REGISTRY)/openmatch-backendapi:$(TAG)
	docker push $(REGISTRY)/openmatch-backendapi:$(ALTERNATE_TAG)

# Deprecated
deprecated-push-mmlogicapi-image: docker deprecated-build-mmlogicapi-image
	docker push $(REGISTRY)/openmatch-mmlogicapi:$(TAG)
	docker push $(REGISTRY)/openmatch-mmlogicapi:$(ALTERNATE_TAG)

# Deprecated
deprecated-push-mmf-go-grpc-serving-simple-image: docker deprecated-build-mmf-go-grpc-serving-simple-image
	docker push $(REGISTRY)/openmatch-mmf-go-grpc-serving-simple:$(TAG)
	docker push $(REGISTRY)/openmatch-mmf-go-grpc-serving-simple:$(ALTERNATE_TAG)

# Deprecated
deprecated-push-backendclient-image: docker deprecated-build-backendclient-image
	docker push $(REGISTRY)/openmatch-backendclient:$(TAG)
	docker push $(REGISTRY)/openmatch-backendclient:$(ALTERNATE_TAG)

# Deprecated
deprecated-push-clientloadgen-image: docker deprecated-build-clientloadgen-image
	docker push $(REGISTRY)/openmatch-clientloadgen:$(TAG)
	docker push $(REGISTRY)/openmatch-clientloadgen:$(ALTERNATE_TAG)

# Deprecated
deprecated-push-frontendclient-image: docker deprecated-build-frontendclient-image
	docker push $(REGISTRY)/openmatch-frontendclient:$(TAG)
	docker push $(REGISTRY)/openmatch-frontendclient:$(ALTERNATE_TAG)

# Deprecated
deprecated-push-evaluator-serving-image: docker deprecated-build-evaluator-serving-image
	docker push $(REGISTRY)/openmatch-evaluator-serving:$(TAG)
	docker push $(REGISTRY)/openmatch-evaluator-serving:$(ALTERNATE_TAG)

# Deprecated
deprecated-build-images: deprecated-build-service-images deprecated-build-client-images deprecated-build-mmf-example-images deprecated-build-evaluator-example-images
deprecated-build-service-images: deprecated-build-frontendapi-image deprecated-build-backendapi-image deprecated-build-mmlogicapi-image
deprecated-build-client-images: deprecated-build-backendclient-image deprecated-build-clientloadgen-image deprecated-build-frontendclient-image
deprecated-build-mmf-example-images: deprecated-build-mmf-go-grpc-serving-simple-image
deprecated-build-evaluator-example-images: deprecated-build-evaluator-serving-image

# Deprecated
deprecated-build-frontendapi-image: docker build-base-build-image
	docker build -f cmd/frontendapi/Dockerfile -t $(REGISTRY)/openmatch-frontendapi:$(TAG) -t $(REGISTRY)/openmatch-frontendapi:$(ALTERNATE_TAG) .

# Deprecated
deprecated-build-backendapi-image: docker build-base-build-image
	docker build -f cmd/backendapi/Dockerfile -t $(REGISTRY)/openmatch-backendapi:$(TAG) -t $(REGISTRY)/openmatch-backendapi:$(ALTERNATE_TAG) .

# Deprecated
deprecated-build-mmlogicapi-image: docker build-base-build-image
	docker build -f cmd/mmlogicapi/Dockerfile -t $(REGISTRY)/openmatch-mmlogicapi:$(TAG) -t $(REGISTRY)/openmatch-mmlogicapi:$(ALTERNATE_TAG) .

# Deprecated
deprecated-build-mmf-go-grpc-serving-simple-image: docker build-base-build-image
	docker build -f examples/functions/golang/grpc-serving/Dockerfile -t $(REGISTRY)/openmatch-mmf-go-grpc-serving-simple:$(TAG) -t $(REGISTRY)/openmatch-mmf-go-grpc-serving-simple:$(ALTERNATE_TAG) .

# Deprecated
deprecated-build-backendclient-image: docker build-base-build-image
	docker build -f examples/backendclient/Dockerfile -t $(REGISTRY)/openmatch-backendclient:$(TAG) -t $(REGISTRY)/openmatch-backendclient:$(ALTERNATE_TAG) .

# Deprecated
deprecated-build-clientloadgen-image: docker build-base-build-image
	docker build -f test/cmd/clientloadgen/Dockerfile -t $(REGISTRY)/openmatch-clientloadgen:$(TAG) -t $(REGISTRY)/openmatch-clientloadgen:$(ALTERNATE_TAG) .

# Deprecated
deprecated-build-frontendclient-image: docker build-base-build-image
	docker build -f test/cmd/frontendclient/Dockerfile -t $(REGISTRY)/openmatch-frontendclient:$(TAG) -t $(REGISTRY)/openmatch-frontendclient:$(ALTERNATE_TAG) .

# Deprecated
deprecated-build-evaluator-serving-image: build-base-build-image
	docker build -f examples/evaluators/golang/serving/Dockerfile -t $(REGISTRY)/openmatch-evaluator-serving:$(TAG) -t $(REGISTRY)/openmatch-evaluator-serving:$(ALTERNATE_TAG) .

deprecated-clean-images: docker
	-docker rmi -f open-match-base-build

	-docker rmi -f $(REGISTRY)/openmatch-frontendapi:$(TAG) $(REGISTRY)/openmatch-frontendapi:$(ALTERNATE_TAG)
	-docker rmi -f $(REGISTRY)/openmatch-backendapi:$(TAG) $(REGISTRY)/openmatch-backendapi:$(ALTERNATE_TAG)
	-docker rmi -f $(REGISTRY)/openmatch-mmlogicapi:$(TAG) $(REGISTRY)/openmatch-mmlogicapi:$(ALTERNATE_TAG)

	-docker rmi -f $(REGISTRY)/openmatch-mmf-go-grpc-serving-simple:$(TAG) $(REGISTRY)/openmatch-mmf-go-grpc-serving-simple:$(ALTERNATE_TAG)

	-docker rmi -f $(REGISTRY)/openmatch-backendclient:$(TAG) $(REGISTRY)/openmatch-backendclient:$(ALTERNATE_TAG)
	-docker rmi -f $(REGISTRY)/openmatch-clientloadgen:$(TAG) $(REGISTRY)/openmatch-clientloadgen:$(ALTERNATE_TAG)
	-docker rmi -f $(REGISTRY)/openmatch-frontendclient:$(TAG) $(REGISTRY)/openmatch-frontendclient:$(ALTERNATE_TAG)
	-docker rmi -f $(REGISTRY)/openmatch-evaluator-serving:$(TAG) $(REGISTRY)/openmatch-evaluator-serving:$(ALTERNATE_TAG)

# Deprecated
deprecated-all-protos: deprecated-golang-protos deprecated-http-proxy-golang-protos deprecated-swagger-json-docs

# Deprecated
deprecated-golang-protos: internal/pb/backend.pb.go internal/pb/frontend.pb.go internal/pb/matchfunction.pb.go internal/pb/messages.pb.go internal/pb/mmlogic.pb.go

# Deprecated
deprecated-http-proxy-golang-protos: internal/pb/backend.pb.gw.go internal/pb/frontend.pb.gw.go internal/pb/matchfunction.pb.gw.go internal/pb/messages.pb.gw.go internal/pb/mmlogic.pb.gw.go

# Deprecated
deprecated-swagger-json-docs: api/protobuf-spec/frontend.swagger.json api/protobuf-spec/backend.swagger.json api/protobuf-spec/mmlogic.swagger.json api/protobuf-spec/matchfunction.swagger.json

# Deprecated
internal/pb/%.pb.go: api/protobuf-spec/%.proto build/toolchain/bin/protoc$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-go$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-grpc-gateway$(EXE_EXTENSION)
	$(PROTOC) $< \
		-I $(REPOSITORY_ROOT) -I $(PROTOC_INCLUDES) \
		--go_out=plugins=grpc:$(REPOSITORY_ROOT)

# Deprecated
internal/pb/%.pb.gw.go: api/protobuf-spec/%.proto internal/pb/%.pb.go build/toolchain/bin/protoc$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-go$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-grpc-gateway$(EXE_EXTENSION)
	$(PROTOC) $< \
		-I $(REPOSITORY_ROOT) -I $(PROTOC_INCLUDES) \
   		--grpc-gateway_out=logtostderr=true,allow_delete_body=true:$(REPOSITORY_ROOT)
# Deprecated
api/protobuf-spec/%.swagger.json: api/protobuf-spec/%.proto internal/pb/%.pb.gw.go build/toolchain/bin/protoc$(EXE_EXTENSION) build/toolchain/bin/protoc-gen-swagger$(EXE_EXTENSION)
	$(PROTOC) $< \
		-I $(REPOSITORY_ROOT) -I $(PROTOC_INCLUDES) --swagger_out=logtostderr=true,allow_delete_body=true:$(REPOSITORY_ROOT)

# Deprecated
# Include structure of the protos needs to be called out do the dependency chain is run through properly.
internal/pb/backend.pb.go: internal/pb/messages.pb.go
internal/pb/frontend.pb.go: internal/pb/messages.pb.go
internal/pb/mmlogic.pb.go: internal/pb/messages.pb.go
internal/pb/matchfunction.pb.go: internal/pb/messages.pb.go

# Deprecated
deprecated-all: deprecated-service-binaries deprecated-client-binaries deprecated-example-binaries tools-binaries
deprecated-service-binaries: cmd/backendapi/backendapi$(EXE_EXTENSION) cmd/frontendapi/frontendapi$(EXE_EXTENSION) cmd/mmlogicapi/mmlogicapi$(EXE_EXTENSION)
deprecated-client-binaries: examples/backendclient/backendclient$(EXE_EXTENSION) test/cmd/clientloadgen/clientloadgen$(EXE_EXTENSION) test/cmd/frontendclient/frontendclient$(EXE_EXTENSION)
deprecated-example-binaries: deprecated-example-mmf-binaries deprecated-example-evaluator-binaries
deprecated-example-mmf-binaries: examples/functions/golang/grpc-serving/grpc-serving$(EXE_EXTENSION)
deprecated-example-evaluator-binaries: examples/evaluators/golang/serving/serving$(EXE_EXTENSION)

# Deprecated
cmd/backendapi/backendapi$(EXE_EXTENSION): internal/pb/backend.pb.go internal/pb/backend.pb.gw.go api/protobuf-spec/backend.swagger.json
	cd cmd/backendapi; $(GO_BUILD_COMMAND)

# Deprecated
cmd/frontendapi/frontendapi$(EXE_EXTENSION): internal/pb/frontend.pb.go internal/pb/frontend.pb.gw.go api/protobuf-spec/frontend.swagger.json
	cd cmd/frontendapi; $(GO_BUILD_COMMAND)

# Deprecated
cmd/mmlogicapi/mmlogicapi$(EXE_EXTENSION): internal/pb/mmlogic.pb.go internal/pb/mmlogic.pb.gw.go api/protobuf-spec/mmlogic.swagger.json
	cd cmd/mmlogicapi; $(GO_BUILD_COMMAND)

# Deprecated
examples/backendclient/backendclient$(EXE_EXTENSION): internal/pb/backend.pb.go
	cd examples/backendclient; $(GO_BUILD_COMMAND)

# Deprecated
examples/evaluators/golang/serving/serving$(EXE_EXTENSION): internal/pb/messages.pb.go
	cd examples/evaluators/golang/serving; $(GO_BUILD_COMMAND)

# Deprecated
examples/functions/golang/grpc-serving/grpc-serving$(EXE_EXTENSION): internal/pb/messages.pb.go
	cd examples/functions/golang/grpc-serving; $(GO_BUILD_COMMAND)

# Deprecated
test/cmd/clientloadgen/clientloadgen$(EXE_EXTENSION):
	cd test/cmd/clientloadgen; $(GO_BUILD_COMMAND)

# Deprecated
test/cmd/frontendclient/frontendclient$(EXE_EXTENSION): internal/pb/frontend.pb.go internal/pb/messages.pb.go
	cd test/cmd/frontendclient; $(GO_BUILD_COMMAND)

deprecated-clean-swagger-docs:
	rm -rf $(REPOSITORY_ROOT)/api/protobuf-spec/*.json

deprecated-clean-protos:
	rm -rf $(REPOSITORY_ROOT)/internal/pb/
	rm -rf $(REPOSITORY_ROOT)/api/protobuf_spec/

deprecated-clean-binaries:
	rm -rf $(REPOSITORY_ROOT)/cmd/minimatch/minimatch
	rm -rf $(REPOSITORY_ROOT)/cmd/backendapi/backendapi
	rm -rf $(REPOSITORY_ROOT)/cmd/frontendapi/frontendapi
	rm -rf $(REPOSITORY_ROOT)/cmd/mmlogicapi/mmlogicapi
	rm -rf $(REPOSITORY_ROOT)/examples/backendclient/backendclient
	rm -rf $(REPOSITORY_ROOT)/examples/evaluators/golang/serving/serving
	rm -rf $(REPOSITORY_ROOT)/examples/functions/golang/grpc-serving/grpc-serving
	rm -rf $(REPOSITORY_ROOT)/test/cmd/clientloadgen/clientloadgen
	rm -rf $(REPOSITORY_ROOT)/test/cmd/frontendclient/frontendclient

run-backendclient: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	$(KUBECTL) run om-backendclient --rm --restart=Never --image-pull-policy=Always -i --tty --image=$(REGISTRY)/openmatch-backendclient:$(TAG) --namespace=$(OPEN_MATCH_KUBERNETES_NAMESPACE) $(KUBECTL_RUN_ENV)

run-frontendclient: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	$(KUBECTL) run om-frontendclient --rm --restart=Never --image-pull-policy=Always -i --tty --image=$(REGISTRY)/openmatch-frontendclient:$(TAG) --namespace=$(OPEN_MATCH_KUBERNETES_NAMESPACE) $(KUBECTL_RUN_ENV)

run-clientloadgen: build/toolchain/bin/kubectl$(EXE_EXTENSION)
	$(KUBECTL) run om-clientloadgen --rm --restart=Never --image-pull-policy=Always -i --tty --image=$(REGISTRY)/openmatch-clientloadgen:$(TAG) --namespace=$(OPEN_MATCH_KUBERNETES_NAMESPACE) $(KUBECTL_RUN_ENV)

# Development Guide

Open Match is a collection of [Go](https://golang.org/) gRPC services that run
within [Kubernetes](https://kubernetes.io).

## Install Prerequisites

To build Open Match you'll need the following applications installed.

 * [Git](https://git-scm.com/downloads)
 * [Go](https://golang.org/doc/install)
 * [Python3 with virtualenv](https://wiki.python.org/moin/BeginnersGuide/Download)
 * Make (Mac: install [XCode](https://itunes.apple.com/us/app/xcode/id497799835))
 * [Docker](https://docs.docker.com/install/) including the
   [post-install steps](https://docs.docker.com/install/linux/linux-postinstall/).

Optional Software

 * [Google Cloud Platform](gcloud.md)
 * [Visual Studio Code](https://code.visualstudio.com/Download) for IDE.
   Vim and Emacs work to.
 * [VirtualBox](https://www.virtualbox.org/wiki/Downloads) recommended for
   [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/).

On Debian-based Linux you can install all the required packages (except Go) by
running:

```bash
sudo apt-get update
sudo apt-get install -y -q python3 python3-virtualenv virtualenv make \
  google-cloud-sdk git unzip tar
```

*It's recommended that you install Go using their instructions because package
managers tend to lag behind the latest Go releases.*

## Get the Code

```bash
# Create a directory for the project.
mkdir -p $HOME/workspace
cd $HOME/workspace
# Download the source code.
git clone https://github.com/googleforgames/open-match.git
cd open-match
# Print the help for the Makefile commands.
make
```

*Typically for contributing you'll want to
[create a fork](https://help.github.com/en/articles/fork-a-repo) and use that
but for purpose of this guide we'll be using the upstream/master.*

## Building

```bash
# Reset workspace
make clean
# Compile all the binaries
make all -j$(nproc)
# Run tests
make test
# Build all the images.
make build-images -j$(nproc)
# Push images to gcr.io (requires Google Cloud SDK installed)
make push-images -j$(nproc)
# Push images to Docker Hub
make REGISTRY=mydockerusername push-images -j$(nproc)
# Generate Kubernetes installation YAML files (Note that the trailing '/' is needed here)
make install/yaml/
```

_**-j$(nproc)** is a flag to tell make to parallelize the commands based on
the number of CPUs on your machine._

## Deploying to Kubernetes

Kubernetes comes in many flavors and Open Match can be used in any of them.

_We support GKE ([setup guide](gcloud.md)), Minikube, and Kubernetes in Docker (KinD) in the Makefile.
As long as kubectl is configured to talk to your Kubernetes cluster as the
default context the Makefile will honor that._

```bash
# Step 1: Create a Kubernetes (k8s) cluster
# KinD cluster: make create-kind-cluster/delete-kind-cluster
# GKE cluster: make create-gke-cluster/delete-gke-cluster
# or create a local Minikube cluster
make create-gke-cluster
# Step 2: Download helm and install Tiller in the cluster
make push-helm
# Step 3: Build and Push Open Match Images to gcr.io
make push-images -j$(nproc)
# Install Open Match in the cluster.
make install-chart

# Create a proxy to Open Match pods so that you can access them locally.
# This command consumes a terminal window that you can kill via Ctrl+C.
# You can run `curl -X POST http://localhost:51504/v1/frontend/tickets` to send
# a DeleteTicket request to the frontend service in the cluster.
# Then try visiting http://localhost:3000/ and view the graphs.
make proxy

# Teardown the install
make delete-chart
```

## Interaction

Before integrating with Open Match you can manually interact with it to get a feel for how it works.

`make proxy-ui` exposes the Swagger UI for Open Match locally on your computer.
You can then go to http://localhost:51500 and view the API as well as interactively call Open Match.

By default you will be talking to the frontend server but you can change the target API url to any of the following:

 * api/frontend.swagger.json
 * api/backend.swagger.json
 * api/synchronizer.swagger.json
 * api/query.swagger.json

For a more current list refer to the api/ directory of this repository. Also matchfunction.swagger.json is not supported.

## IDE Support

Open Match is a standard Go project so any IDE that understands that should
work. We use [Go Modules](https://github.com/golang/go/wiki/Modules) which is a
relatively new feature in Go so make sure the IDE you are using was built around
Summer 2019. The latest version of
[Visual Studio Code](https://code.visualstudio.com/download) supports it.

If your IDE is too old you can create a
[Go workspace](https://golang.org/doc/code.html#Workspaces).

```bash
# Create the Go workspace in $HOME/workspace/ directory.
mkdir -p $HOME/workspace/src/open-match.dev/
cd $HOME/workspace/src/open-match.dev/
# Download the source code.
git clone https://github.com/googleforgames/open-match.git
cd open-match
export GOPATH=$HOME/workspace/
```

## Pull Requests

If you want to submit a Pull Request there's some tools to help prepare your
change.

```bash
# Runs code generators, tests, and linters.
make presubmit
```

`make presubmit` catches most of the issues your change can run into. If the
submit checks fail you can run it locally via,

```bash
make local-cloud-build
```

Our [continuous integration](https://console.cloud.google.com/cloud-build/builds?project=open-match-build)
runs against all PRs. In order to see your build results you'll need to
become a member of
[open-match-discuss@googlegroups.com](https://groups.google.com/forum/#!forum/open-match-discuss).


## Makefile

The Makefile is the core of Open Match's build process. There's a lot of
commands but here's a list of the important ones and patterns to remember them.

```bash
# Help
make

# Reset workspace (delete all build artifacts)
make clean
# Delete auto-generated protobuf code and swagger API docs.
make clean-protos clean-swagger-docs
# make clean-* deletes some part of the build outputs.

# Build all Docker images
make build-images
# Build frontend docker image.
make build-frontend-image

# Formats, Vets, and tests the codebase.
make fmt vet test
# Same as above also regenerates autogen files.
make presubmit

# Run website on http://localhost:8080
make run-site

# Proxy all Open Match processes to view them.
make proxy
```

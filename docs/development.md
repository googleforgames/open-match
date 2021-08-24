# Development Guide

Open Match is a collection of [Go](https://golang.org/) gRPC services that run
within [Kubernetes](https://kubernetes.io).

## Install Prerequisites

To build Open Match you'll need the following applications installed.

 * [Git](https://git-scm.com/downloads)
 * [Go](https://golang.org/doc/install)
 * Make (Mac: install [XCode](https://itunes.apple.com/us/app/xcode/id497799835))
 * [Docker](https://docs.docker.com/install/) including the
   [post-install steps](https://docs.docker.com/install/linux/linux-postinstall/).

Optional Software

 * [Visual Studio Code](https://code.visualstudio.com/Download) for IDE.
   Vim and Emacs work to.
 * [VirtualBox](https://www.virtualbox.org/wiki/Downloads) recommended for
   [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/).

On Debian-based Linux you can install all the required packages (except Go) by
running:

```bash
sudo apt-get update
sudo apt-get install -y -q make google-cloud-sdk git unzip tar
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
but for purpose of this guide we'll be using the upstream/main.*

## Building code and images

```bash
# Reset workspace
make clean
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
# Step 2: Build and Push Open Match Images to gcr.io
make push-images -j$(nproc)
# Step 3: Install Open Match in the cluster.
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

## Iterating
While iterating on the project, you may need to:
1. Install/Run everything
2. Make some code changes
3. Make sure the changes compile by running `make test`
4. Build and push Docker images to your personal registry by running `make push-images -j$(nproc)`
5. Deploy the code change by running `make install-chart`
6. Verify it's working by [looking at the logs](#accessing-logs) or looking at the monitoring dashboard by running `make proxy-grafana`
7. Tear down Open Match by running `make delete-chart`

## Accessing logs
To look at Open Match core services' logs, run:
```bash
# Replace open-match-frontend with the service name that you would like to access
kubectl logs -n open-match svc/open-match-frontend
```

## API References
While integrating with Open Match you may want to understand its API surface concepts or interact with it and get a feel for how it works.

The APIs are defined in `proto` format under the `api/` folder, with references available at [open-match.dev](https://open-match.dev/site/docs/reference/api/).

You can also run `make proxy-ui` to exposes the Swagger UI for Open Match locally on your computer after [deploying it to Kubernetes](#deploying-to-kubernetes), then go to http://localhost:51500 and view the REST APIs as well as interactively call Open Match.

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

If you want to submit a Pull Request, `make presubmit` can catch most of the issues your change can run into.

Our [continuous integration](https://console.cloud.google.com/cloud-build/builds?project=open-match-build)
runs against all PRs. In order to see your build results you'll need to
become a member of
[open-match-discuss@googlegroups.com](https://groups.google.com/forum/#!forum/open-match-discuss).

## Building

Documentation and usage guides on how to set up and customize Open Match.

### Precompiled container images

Once we reach a 1.0 release, we plan to produce publicly available (Linux) Docker container images of major releases in a public image registry. Until then, refer to the 'Compiling from source' section below.

### Compiling from source

The easiest way to build Open Match is to use the Makefile. Before you can use the Makefile make sure you have the following dependencies:

```bash
# Install Open Match Toolchain Dependencies (Debian other OSes including Mac OS X have similar dependencies)
sudo apt-get update; sudo apt-get install -y -q python3 python3-virtualenv virtualenv make google-cloud-sdk git unzip tar
# Setup your repository like Go workspace, https://golang.org/doc/code.html#Workspaces
# This requirement will go away soon.
mkdir -p workspace/src/github.com/GoogleCloudPlatform/
cd workspace/src/github.com/GoogleCloudPlatform/
export GOPATH=$HOME/workspace
export GO111MODULE=on
git clone https://github.com/GoogleCloudPlatform/open-match.git
cd open-match
```

[Docker](https://docs.docker.com/install/) and [Go 1.11+](https://golang.org/dl/) is also required. If your distro is new enough you can probably run `sudo apt-get install -y golang` or download the newest version from https://golang.org/.

To build all the artifacts of Open Match you can simply run the following commands.

```bash
# Downloads all the tools needed to build Open Match
make install-toolchain
# Generates protocol buffer code files
make all-protos
# Builds all the binaries
make all
# Builds all the images.
make build-images
```

Once build you can use a command like `docker images` to see all the images that were build.

Before creating a pull request you can run `make local-cloud-build` to simulate a Cloud Build run to check for regressions.

The directory structure is a typical Go structure so if you do the following you should be able to work on this project within your IDE.

```bash
cd $GOPATH
mkdir -p src/github.com/GoogleCloudPlatform/
cd src/github.com/GoogleCloudPlatform/
# If you're going to contribute you'll want to fork open-match, see CONTRIBUTING.md for details.
git clone https://github.com/GoogleCloudPlatform/open-match.git
cd open-match
# Open IDE in this directory.
```

Lastly, this project uses go modules so you'll want to set `export GO111MODULE=on` before building.

## Zero to Open Match
To deploy Open Match quickly to a Kubernetes cluster run these commands.

```bash
# Downloads all the tools.
make install-toolchain
# Create a GKE Cluster
make create-gke-cluster
# OR Create a Minikube Cluster
make create-mini-cluster
# Install Helm
make push-helm
# Build and push images
make push-images -j4
# Deploy Open Match with example functions
make install-chart install-example-chart
```

## Docker Image Builds

All the core components for Open Match are written in Golang and use the [Dockerfile multistage builder pattern](https://docs.docker.com/develop/develop-images/multistage-build/). This pattern uses intermediate Docker containers as a Golang build environment while producing lightweight, minimized container images as final build artifacts. When the project is ready for production, we will modify the `Dockerfile`s to uncomment the last build stage. Although this pattern is great for production container images, it removes most of the utilities required to troubleshoot issues during development.

## Configuration
Currently, each component reads a local config file `matchmaker_config.json`, and all components assume they have the same configuration. To this end, there is a single centralized config file located in the `<REPO_ROOT>/config/` which is symlinked to each component's subdirectory for convenience when building locally. When `docker build`ing the component container images, the Dockerfile copies the centralized config file into the component directory.

We plan to replace this with a Kubernetes-managed config with dynamic reloading, please join the discussion in [Issue #42](issues/42). 

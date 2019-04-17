![Open Match](site/assets/omlogo.png)

[![GoDoc](https://godoc.org/github.com/GoogleCloudPlatform/open-match?status.svg)](https://godoc.org/github.com/GoogleCloudPlatform/open-match)
[![Go Report Card](https://goreportcard.com/badge/github.com/GoogleCloudPlatform/open-match)](https://goreportcard.com/report/github.com/GoogleCloudPlatform/open-match)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/GoogleCloudPlatform/open-match/blob/master/LICENSE)

Open Match is an open source game matchmaking framework designed to allow game creators to build matchmakers of any size easily and with as much possibility for sharing and code re-use as possible. It’s designed to be flexible, extensible, and scalable.

Matchmaking begins when a player tells the game that they want to play. Every player has a set of attributes like skill, location, playtime, win-lose ratio, etc which may factor in how they are paired with other players. Typically, there's a trade off between the quality of the match vs the time to wait. Since Open Match is designed to scale with the player population, it should be possible to still have high quality matches while having high player count.

Under the covers matchmaking approaches touch on significant areas of computer science including graph theory and massively concurrent processing. Open Match is an effort to provide a foundation upon which these difficult problems can be addressed by the wider game development community. As Josh Menke &mdash; famous for working on matchmaking for many popular triple-A franchises &mdash; put it:

["Matchmaking, a lot of it actually really is just really good engineering. There's a lot of really hard networking and plumbing problems that need to be solved, depending on the size of your audience."](https://youtu.be/-pglxege-gU?t=830)

This project attempts to solve the networking and plumbing problems, so game developers can focus on the logic to match players into great games.

## Running Open Match
Open Match framework is a collection of servers that run within Kubernetes (the [puppet master](https://en.wikipedia.org/wiki/Puppet_Master_(gaming)) for your server cluster.)


## Deploy to Kubernetes

If you have an [existing Kubernetes cluster](https://cloud.google.com/kubernetes-engine/docs/how-to/creating-a-cluster) you can run these commands to install Open Match.

```bash
# Grant yourself cluster-admin permissions so that you can deploy service accounts.
kubectl create clusterrolebinding myname-cluster-admin-binding --clusterrole=cluster-admin --user=$(YOUR_KUBERNETES_USER_NAME)
# Place all Open Match components in their own namespace.
kubectl create namespace open-match
# Install Open Match and monitoring services.
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/open-match/master/install/yaml/install.yaml --namespace open-match
# Install the example MMF and Evaluator.
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/open-match/master/install/yaml/install-example.yaml --namespace open-match
```

To delete Open Match

```bash
# Delete the open-match namespace that holds all the Open Match configuration.
kubectl delete namespace open-match
```

## Development
Open Match can be deployed locally or in the cloud for development. Below are the steps to build, push, and deploy the binaries to Kubernetes.

### Deploy to Minikube (Locally)
[Minikube](https://kubernetes.io/docs/setup/minikube/) is Kubernetes in a VM. It's mainly used for development.

```bash
# Create a Minikube Cluster and install Helm
make create-mini-cluster push-helm
# Deploy Open Match with example functions
make REGISTRY=gcr.io/open-match-public-images TAG=latest install-chart install-example-chart
```

### Deploy to Google Cloud Platform (Cloud)

Create a GCP project via [Google Cloud Console](https://console.cloud.google.com/). Billing must be enabled but if you're a new customer you can get some [free credits](https://cloud.google.com/free/). When you create a project you'll need to set a Project ID, if you forget it you can see it here, https://console.cloud.google.com/iam-admin/settings/project.

Now install [Google Cloud SDK](https://cloud.google.com/sdk/) which is the command line tool to work against your project. The following commands log you into your GCP Project.

```bash
# Login to your Google Account for GCP.
gcloud auth login
gcloud config set project $YOUR_GCP_PROJECT_ID
# Enable GCP services
gcloud services enable containerregistry.googleapis.com
gcloud services enable container.googleapis.com
# Test that everything is good, this command should work.
gcloud compute zone list
```

Once everything is setup you can deploy Open Match by creating a cluster in Google Kubernetes Engine (GKE).

```bash
# Create a GKE Cluster and install Helm
make create-gke-cluster push-helm
# Deploy Open Match with example functions
make REGISTRY=gcr.io/open-match-public-images TAG=latest install-chart install-example-chart
```

Once deployed you can view the jobs in [Cloud Console](https://console.cloud.google.com/kubernetes/workload).

### Compiling From Source

The easiest way to build Open Match is to use the [Makefile](Makefile). Before you can use the Makefile make sure you have the following dependencies:

```bash
# Install Open Match Toolchain Dependencies (for Debian, other OSes including Mac OS X have similar dependencies)
sudo apt-get update; sudo apt-get install -y -q python3 python3-virtualenv virtualenv make google-cloud-sdk git unzip tar
# Setup your repository like Go workspace, https://golang.org/doc/code.html#Workspaces
# This requirement will go away soon.
mkdir -p $HOME/workspace/src/github.com/GoogleCloudPlatform/
cd $HOME/workspace/src/github.com/GoogleCloudPlatform/
export GOPATH=$HOME/workspace
export GO111MODULE=on
git clone https://github.com/GoogleCloudPlatform/open-match.git
cd open-match
```

[Docker](https://docs.docker.com/install/) and [Go 1.12+](https://golang.org/dl/) is also required.

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

Lastly, this project uses go modules so you'll want to set `export GO111MODULE=on` in your `~/.bashrc`.

The [Build Queue](https://console.cloud.google.com/cloud-build/builds?project=open-match-build) runs against all PRs, requires membership to [open-match-discuss@googlegroups.com](https://groups.google.com/forum/#!forum/open-match-discuss).

## Support

* [Slack Channel](https://open-match.slack.com/) ([Signup](https://join.slack.com/t/open-match/shared_invite/enQtNDM1NjcxNTY4MTgzLWQzMzE1MGY5YmYyYWY3ZjE2MjNjZTdmYmQ1ZTQzMmNiNGViYmQyN2M4ZmVkMDY2YzZlOTUwMTYwMzI1Y2I2MjU))
* [File an Issue](https://github.com/GoogleCloudPlatform/open-match/issues/new)
* [Mailing list](https://groups.google.com/forum/#!forum/open-match-discuss)
* [Managed Service Survey](https://goo.gl/forms/cbrFTNCmy9rItSv72)

## Contributing

Please read the [contributing](CONTRIBUTING.md) guide for directions on submitting Pull Requests to Open Match.

See the [Development guide](docs/development.md) for documentation for development and building Open Match from source.

The [Release Process](docs/governance/release_process.md) documentation displays the project's upcoming release calendar and release process.

Open Match is in active development - we would love your help in shaping its future!

## Documentation

For more information on the technical underpinnings of Open Match you can refer to the [docs/](docs/) directory. 

## Code of Conduct

Participation in this project comes under the [Contributor Covenant Code of Conduct](code-of-conduct.md)

## Disclaimer
This software is currently alpha, and subject to change. Although Open Match has already been used to run [production workloads within Google](https://cloud.google.com/blog/topics/inside-google-cloud/no-tricks-just-treats-globally-scaling-the-halloween-multiplayer-doodle-with-open-match-on-google-cloud), but it's still early days on the way to our final goal. There's plenty left to write and we welcome contributions. **We strongly encourage you to engage with the community through the [Slack or Mailing lists](#support) if you're considering using Open Match in production before the 1.0 release, as the documentation is likely to lag behind the latest version a bit while we focus on getting out of alpha/beta as soon as possible.**

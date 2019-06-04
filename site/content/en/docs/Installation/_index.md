---
title: "Install Open Match in Kubernetes"
linkTitle: "Installation"
weight: 1
description: >
  Follow this guide to install Open Match in your Kubernetes cluster.
---

In this quickstart, we'll create a Kubernetes cluster, install Open Match, and create matches with the example tools.

# Setup Kubernetes

This guide is for users that do not have a Kubernetes cluster. If you already have one that you can install Open Match into skip this section.

* [Set up a Google Cloud Kubernetes Cluster]({{< relref "./setup-gke.md" >}}) (*this may involve extra charges unless you are on free tier*)
* [Set up a Local Minikube cluster](https://kubernetes.io/docs/setup/minikube/)

## Install Open Match Servers

The simplest way to install Open Match is to use the install.yaml files for the latest release.
This installs Open Match with the default configuration.

```bash
# Create a cluster role binding (if using gcloud on Linux or OSX)
kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user `gcloud config get-value account`

# Create a cluster role binding (if using gcloud on Windows)
for /F %i in ('gcloud config get-value account') do kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user %i

# Create a cluster role binding (if using minikube)
kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --serviceaccount=kube-system:default

# Create a namespace to place all the Open Match components in.
kubectl create namespace open-match

# Install the core Open Match and monitoring services.
kubectl apply -f https://github.com/googleforgames/open-match/releases/download/v0.5.0/install.yaml --namespace open-match
```

### Install Example Components

Open Match framework requires the user to author a custom match function and an evaluator that are invoked to create matches. For demo purposes, we will use an example MMF and Evaluator. The following command deploys these in the kubernetes cluster:

```bash
# Install the example MMF and Evaluator.
kubectl apply -f https://github.com/googleforgames/open-match/releases/download/v0.5.0/install-example.yaml --namespace open-match
```

This command also deploys a component that continuously generates players with different properties and adds them to Open Match state storage. This is because a populated player pool is required to generate matches.

### Generate Matches

In a real setup, a game backend (Director / DGS etc.) will request Open Match for matches. For demo purposes, this is simulated by a backend client that requests Open Match to continuously list matches till it runs out of players.

```bash
# Install the example MMF and Evaluator.
kubectl run om-backendclient --rm --restart=Never --image-pull-policy=Always -i --tty --image=gcr.io/open-match-public-images/openmatch-backendclient:0.5.0 --namespace=open-match
```

If successful, the backend client should successfully generate matches, displaying players populated in Rosters.

### Delete Open Match

To delete Open Match from this cluster, simply run:

```bash
kubectl delete namespace open-match
```

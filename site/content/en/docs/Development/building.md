---
title: "Building"
linkTitle: "Building"
date: 2017-01-05
description: >
  You can build Open Match quickly with just a few tools.
---

To build Open Match we'll first need to setup your local environment. Installing the following:
 * Docker
 * Go
 * Make

Once those are installed you can build Open Match with the following commands.

```bash
make install-toolchain
make all-protos
make all
make test
```

# Deployment

It's also easy to setup a Kubernetes cluster and install Open Match into it. Simply run the following commands.

```bash
# For Minikube Cluster
make create-mini-cluster
make push-helm

# For GKE Cluster
export GCP_PROJECT_ID=$YOUR_PROJECT_ID
make create-gke-cluster
make push-helm

# Build Images
make push-images

# Deploy to the cluster.
make install-chart
make install-example-chart

# View the Kubernetes Dashboard once installed.
make proxy-dashboard

# View Prometheus and Grafana
make proxy-prometheus
make proxy-grafana
```

# Teardown
```bash
# For Minikube
make delete-mini-cluster

# For GKE
make delete-gke-cluster

# Remote Helm Charts
make delete-example-chart
make delete-chart
```

# Push New Changes
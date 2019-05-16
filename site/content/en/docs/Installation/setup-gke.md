---
title: "Setup Google Kubernetes Engine"
linkTitle: "Setup GKE"
weight: 1
description: >
  Follow these steps to create a Google Kubernetes Engine (GKE) cluster in Google Cloud Platform (GCP).
---
# Create a GKE Cluster

Below are the steps to create a GKE cluster in Google Cloud Platform.

* Create a GCP project via [Google Cloud Console](https://console.cloud.google.com/).
* Billing must be enabled. If you're a new customer you can get some [free credits](https://cloud.google.com/free/).
* When you create a project you'll need to set a Project ID, if you forget it you can see it here, https://console.cloud.google.com/iam-admin/settings/project.
* Install [Google Cloud SDK](https://cloud.google.com/sdk/) which is the command line tool to work against your project.

Here are the next steps using the gcloud tool.  

```bash
# Login to your Google Account for GCP
gcloud auth login
gcloud config set project $YOUR_GCP_PROJECT_ID

# Enable necessary GCP services
gcloud services enable containerregistry.googleapis.com
gcloud services enable container.googleapis.com

# Enable optional GCP services for security hardening
gcloud services enable containeranalysis.googleapis.com
gcloud services enable binaryauthorization.googleapis.com

# Test that everything is good, this command should work.
gcloud compute zones list

# Create a GKE Cluster in this project
gcloud container clusters create --machine-type n1-standard-2 open-match-dev-cluster --zone us-west1-a --tags open-match
```

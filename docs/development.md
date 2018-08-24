## Compiling from source

All components of Open Match produce (Linux) Docker container images as artifacts, and there are included `Dockerfile`s for each. [Google Cloud Platform Cloud Build](https://cloud.google.com/cloud-build/docs/) users will also find `cloudbuild_<name>.yaml` files for each component in the repository root.

Note: Although Google Cloud Platform includes some free usage, you may incur charges following this guide if you use GCP products.

## Example of building using Google Cloud Builder

The [Quickstart for Docker](https://cloud.google.com/cloud-build/docs/quickstart-docker) guide explains how to set up a project, enable billing, enable Cloud Build, and install the Cloud SDK if you haven't do these things before. Once you get to 'Preparing source files' you are ready to continue with the steps below.

* Clone this repo to a local machine or Google Cloud Shell session, and cd into it.
* Run the following one-line bash script to compile all the images for the first time, and push them to your gcr.io registry. You must enable the [Container Registry API](https://console.cloud.google.com/flows/enableapi?apiid=containerregistry.googleapis.com) first.
```
for dfile in $(ls Dockerfile.*); do gcloud builds submit --substitutions TAG_NAME=dev --config cloudbuild_${dfile##*.}.yaml; done
```

## Example of starting a GKE cluster

A cluster with mostly default settings will work for this development guide.  In the Cloud SDK command below we start it with machines that have 4 vCPUs.  Alternatively, you can use the 'Create Cluster' button in [Google Cloud Console]("https://console.cloud.google.com/kubernetes").

```
gcloud container clusters create --machine-type n1-standard-4 open-match-dev-cluster
```

## Configuration

Currently, each component reads a local config file `matchmaker_config.json` , and all components assume they have the same configuration.  To this end, there is a single centralized config file located in the `<REPO_ROOT>/config/` which is symlinked to each component's subdirectory for convenience when building locally. When `docker build`ing the component container images, the Dockerfile copies the centralized config file into the component directory. 

We plan to replace this with a Kubernetes-managed config with dynamic reloading when development time allows.  Pull requests are welcome!

## Development and contribution

The rest of this guide assumes you have a cluster (example is using GKE, but works on any cluster with a little tweaking), and kubectl configured to administer it, and you've built all the Docker container images described by `Dockerfiles` in the repository root directory. If you have already have a cluster, 

**NOTE** Kubernetes resources that use container images will need to be updated with **your container registry URI**. Here's an example command in Linux to do this (just replace YOUR_REGISTRY_URI with the appropriate location in your environment):
```
sed -i 's|gcr.io/matchmaker-dev|YOUR_REGISTRY_URI|g' *deployment.json
```
If you are using the gcr.io registry on GCP, the default URI is `gcr.io/<PROJECT_NAME>`. 

All of the Kubernetes pod specs in the provided JSON files are looking for images tagged 'dev', which matches the output of the Google Cloud Build command above. If you aren't building your Open Match components using the provided cloudbuild_COMPONENT.yaml files, be sure to tag them appropriately or edit the Kubernetes deployments.

* Start a copy of redis and a service in front of it:
```
kubectl apply -f k8s/redis_deployment.json
kubectl apply -f k8s/redis_service.json
```
* In order to kick off jobs, the matchmaker function orchestrator needs a service account with permission to administer the cluster. This should be updated to have min required perms before launch, this is pretty permissive but acceptable for closed testing:
```
kubectl apply -f k8s/serviceaccountperms.json
```
* Run the **core components**: the frontend API, the backend API, and the matchmaker function orchestrator (MMFOrc). 
```
kubectl apply -f k8s/backendapi_deployment.json
kubectl apply -f k8s/backendapi_service.json
kubectl apply -f k8s/frontendapi_deployment.json
kubectl apply -f k8s/frontendapi_service.json
kubectl apply -f k8s/mmforc_deployment.json
```
* [optional, but recommended] Configure the OpenCensus metrics services:
```
kubectl apply -f k8s/metrics_services.json
```
* [optional] Trying to apply the Kubernetes Prometheus Operator resource definition files without a cluster-admin rolebinding on GKE doesn't work without running the following command first. See https://github.com/coreos/prometheus-operator/issues/357
```
kubectl create clusterrolebinding projectowner-cluster-admin-binding --clusterrole=cluster-admin --user=<GCP_ACCOUNT>
```
* [optional, beta] If using Prometheus as your metrics gathering backend, configure the [Prometheus Kubernetes Operator](https://github.com/coreos/prometheus-operator):

```
kubectl apply -f k8s/promoper.json
kubectl apply -f k8s/prometheus.json
kubectl apply -f k8s/prometheus_service.json
kubectl apply -f k8s/metrics_servicemonitors.json
```
You should now be able to see the core component pods running using a `kubectl get pods`, and the core component metrics in the Prometheus Web UI by running `kubectl proxy <PROMETHEUS_POD_NAME> 9090:9090` in your local shell, then opening http://localhost:9090/targets in your browser to see which services Prometheus is collecting from.


# Open Match integrations

## Structured logging

Logging for Open Match uses the [Golang logrus module](https://github.com/sirupsen/logrus) to provide structured logs (https://github.com/sirupsen/logrus).  Logs are output to `stdout` in each component, as expected by Docker and Kubernetes. If you have a specific log aggregator as your final destination, we recommend you have a look at the logrus documentation as there is probably a log formatter that plays nicely with your stack.

## Instrumentation for metrics

Open Match uses http://opencensus.io for metrics instrumentation.  gRPC integrations are built-in, and Golang redigo module integrations are incoming (but haven't been merged into the official repo, see https://github.com/opencensus-integrations/redigo/pull/1).  All of the core components expose HTTP `/metrics` endpoints on the port defined in `config/matchmaker_config.json` (default: 9555) for Prometheus to scrape. If you would like to export to a different metrics aggregation platform, we suggest you have a look at the OpenCensus documentation - there may be one written for you already, and switching to it may be as simple as changing a few lines of code.

A standard for instrumentation of MMFs is currently in planning.  

## Redis setup

By default, Open Match expects you to run Redis *somewhere*.  `Host:port` connection information can be put in the config file for any Redis instance reachable from the [Kubernetes namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) (by default, Open Match sensibly runs in the Kubernetes `default` namespace). In most instances, we expect users will run a copy of Redis in a pod in Kubernetes, with a service pointing to it.  

* Basic auth for Redis instances isn't implemented (but could be, trivially).
* HA configurations for Redis aren't implemented by the provided Kubernetes resource definition files, but Open Match expects the Redis service to be named `redis-sentinel`, which provides an easier path to multi-instance deployments.

# Missing functionality

* Player/Group records generated when a client enters the matchmaking pool need to be removed after a certain amount of time with no activity. When using Redis, this will be implemented as a expiration on the player record.
* Names of the containers to be run for the custom matchmaking functions should be optionally overridden by the matchmaking profile. 
* Instrumentation of MMFs is in the planning stages.  Since MMFs are by design meant to be completely customizable (to the point of allowing any process that can be packaged in a Docker container), metrics/stats will need to have an expected format and outgoing pathway formed.  Currently the thought is that it might be that the metrics should be written to a particular key in statestorage in a format compatible with opencensus, and will be collected, aggreggated, and exported to Prometheus in another process. 
* The Kubernetes service account used by the MMFOrc should be updated to have min required permissions.
* Autoscaling isn't turned on for the Frontend or Backend API Kubernetes deployments by default.
* Match profiles should be able to define multiple MMF container images to run, but this is not currently supported. This enables A/B testing and several other scenarios.
* Out-of-the-box, the Redis deployment should be a HA configuration using Redis seninel.

# Planned improvements

* “Writing your first matchmaker” getting started guide will be included in an upcoming version.
* Documentation for using the example customizable components and the `backendstub` and `frontendstub` applications to do an end-to-end (e2e) test will be written. This all works now, but needs to be written up.
* A [Helm](https://helm.sh/) chart to stand up Open Match will be provided in an upcoming version.
* We plan to host 'official' docker images for all release versions of the core components in publicly available docker registries soon.
* CI/CD for this repo and the associated status tags are planned.
* [OpenCensus tracing](https://opencensus.io/core-concepts/tracing/) will be implemented in an upcoming version.
* Read logrus logging configuration from matchmaker_config.json.
* Golang unit tests will be shipped in an upcoming version.
* All state storage operations should be isolated from core components into the `statestorage/` modules.  This is necessary precursor work to enabling Open Match state storage to use software other than Redis.
* The MMFOrc component name will be updated in a future version to something easier to understand.  Suggestions welcome!

# FAQs

(this should link to/be replaced by to the official FAQ)
1. **"I notice that all the APIs use gRPC. What if I want to make my calls using REST, or via a Websocket?"** (gateway/proxy OSS projects are available)

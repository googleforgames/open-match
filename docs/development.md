# Compiling from source

All components of Open Match produce (Linux) Docker container images as artifacts, and there are included `Dockerfile`s for each. [Google Cloud Platform Cloud Build](https://cloud.google.com/cloud-build/docs/) users will also find `cloudbuild_<name>.yaml` files for each component in the repository root.

Note: Although Google Cloud Platform includes some free usage, you may incur charges following this guide if you use GCP products.

**This project has not completed a first-line security audit, and there are definitely going to be some service accounts that are too permissive.  This should be fine for testing/development in a local environment, but absolutely should not be used as-is in a production environment.**

## Example of building using Google Cloud Builder

The [Quickstart for Docker](https://cloud.google.com/cloud-build/docs/quickstart-docker) guide explains how to set up a project, enable billing, enable Cloud Build, and install the Cloud SDK if you haven't do these things before. Once you get to 'Preparing source files' you are ready to continue with the steps below.

* Clone this repo to a local machine or Google Cloud Shell session, and cd into it.
* Run the following one-line bash script to compile all the images for the first time, and push them to your gcr.io registry. You must enable the [Container Registry API](https://console.cloud.google.com/flows/enableapi?apiid=containerregistry.googleapis.com) first.
```
for dfile in $(ls Dockerfile.*); do gcloud builds submit --config cloudbuild_${dfile##*.}.yaml; done
```

## Example of starting a GKE cluster

A cluster with mostly default settings will work for this development guide.  In the Cloud SDK command below we start it with machines that have 4 vCPUs.  Alternatively, you can use the 'Create Cluster' button in [Google Cloud Console]("https://console.cloud.google.com/kubernetes").

```
gcloud container clusters create --machine-type n1-standard-4 open-match-dev-cluster
```

## Configuration

Currently, each component reads a local config file `matchmaker_config.json` , and all components assume they have the same configuration.  To this end, there is a single centralized config file located in the `<REPO_ROOT>/config/` which is symlinked to each component's subdirectory for convenience when building locally. 

We plan to replace this with a Kubernetes-managed config with dynamic reloading when development time allows.  Pull requests are welcome!

## Running Open Match in a development environment 

The rest of this guide assumes you have a cluster (example is using GKE, but works on any cluster with a little tweaking), and kubectl configured to administer that cluster, and you've built all the Docker container images described by `Dockerfiles` in the repository root directory and given them the docker tag 'dev'.  It assumes you are in the `<REPO_ROOT>/deployments/k8s/` directory.

**NOTE** Kubernetes resources that use container images will need to be updated with **your container registry URI**. Here's an example command in Linux to do this (just replace YOUR_REGISTRY_URI with the appropriate location in your environment):
```
sed -i 's|gcr.io/matchmaker-dev|YOUR_REGISTRY_URI|g' *deployment.json
```
If you are using the gcr.io registry on GCP, the default URI is `gcr.io/<PROJECT_NAME>`. 

* Start a copy of redis and a service in front of it:
```
kubectl apply -f redis_deployment.json
kubectl apply -f redis_service.json
```
* Run the **core components**: the frontend API, the backend API, and the matchmaker function orchestrator (MMFOrc). 
**NOTE** In order to kick off jobs, the matchmaker function orchestrator needs a service account with permission to administer the cluster. This should be updated to have min required perms before launch, this is pretty permissive but acceptable for closed testing:
```
kubectl apply -f backendapi_deployment.json
kubectl apply -f backendapi_service.json
kubectl apply -f frontendapi_deployment.json
kubectl apply -f frontendapi_service.json
kubectl apply -f mmforc_deployment.json
kubectl apply -f mmforc_serviceaccount.json
```
* [optional, but recommended] Configure the OpenCensus metrics services:
```
kubectl apply -f metrics_services.json
```
* [optional] Trying to apply the Kubernetes Prometheus Operator resource definition files without a cluster-admin rolebinding on GKE doesn't work without running the following command first. See https://github.com/coreos/prometheus-operator/issues/357
```
kubectl create clusterrolebinding projectowner-cluster-admin-binding --clusterrole=cluster-admin --user=<GCP_ACCOUNT>
```
* [optional, uses beta software] If using Prometheus as your metrics gathering backend, configure the [Prometheus Kubernetes Operator](https://github.com/coreos/prometheus-operator):

```
kubectl apply -f prometheus_operator.json
kubectl apply -f prometheus.json
kubectl apply -f prometheus_service.json
kubectl apply -f metrics_servicemonitor.json
```
You should now be able to see the core component pods running using a `kubectl get pods`, and the core component metrics in the Prometheus Web UI by running `kubectl proxy <PROMETHEUS_POD_NAME> 9090:9090` in your local shell, then opening http://localhost:9090/targets in your browser to see which services Prometheus is collecting from.

### End-to-End testing

**Note** The programs provided below are just bare-bones manual testing programs with no automation and no claim of code coverage. This sparseness of this part of the documentation is because we expect to discard all of these tools and write a fully automated end-to-end test suite and a collection of load testing tools, with extensive stats output and tracing capabilities before 1.0 release. Tracing has to be integrated first, which will be in an upcoming release.

In the end: *caveat emptor*. These tools all work and are quite small, and as such are fairly easy for developers to understand by looking at the code and logging output. They are provided as-is just as a reference point of how to begin experimenting with Open Match integrations.

* `examples/frontendclient` is a fake client for the Frontend API.  It pretends to be a real game client connecting to Open Match and requests a game, then dumps out the connection string it receives.  Note that it doesn't actually test the return path by looking for arbitrary results from your matchmaking function; it pauses and tells you the name of a key to set a connection string in directly using a redis-cli client.
* `examples/backendclient` is a fake client for the Backend API.  It pretends to be a dedicated game server backend connecting to openmatch and sending in a match profile to fill.  Once it receives a match object with a roster, it will also issue a call to assign the player IDs, and gives an example connection string.  If it never seems to get a match, make sure you're adding players to the pool using the other two tools.
* `test/cmd/client` is a (VERY) basic client load simulation tool.  It does **not** test the Frontend API - in fact, it ignores it and writes players directly to state storage on its own.  It doesn't do anything but loop endlessly, writing players into state storage so you can test your backend integration, and run your custom MMFs and Evaluators (which are only triggered when there are players in the pool).

### Resources

* [Prometheus Operator spec](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md)


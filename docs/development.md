# Compiling from source

All components of Open Match produce (Linux) Docker container images as artifacts, and there are included `Dockerfile`s for each. [Google Cloud Platform Cloud Build](https://cloud.google.com/cloud-build/docs/) users will also find `cloudbuild_<name>.yaml` files for each component in the repository root.

Note: Although Google Cloud Platform includes some free usage, you may incur charges following this guide if you use GCP products.

**This project has not completed a first-line security audit, and there are definitely going to be some service accounts that are too permissive.  This should be fine for testing/development in a local environment, but absolutely should not be used as-is in a production environment without your team/organization evaluating it's permissions.**

## Example of building using Google Cloud Builder

The [Quickstart for Docker](https://cloud.google.com/cloud-build/docs/quickstart-docker) guide explains how to set up a project, enable billing, enable Cloud Build, and install the Cloud SDK if you haven't do these things before. Once you get to 'Preparing source files' you are ready to continue with the steps below.

* Clone this repo to a local machine or Google Cloud Shell session, and cd into it.
* In Linux, you can run the following one-line bash script to compile all the images for the first time, and push them to your gcr.io registry. You must enable the [Container Registry API](https://console.cloud.google.com/flows/enableapi?apiid=containerregistry.googleapis.com) first.
    ```
    for dfile in $(ls Dockerfile.*); do gcloud builds submit --config cloudbuild_${dfile##*.}.yaml & done
    ```
* Once the cloud builds have completed, you can verify that all the builds succeeded by checking the list of images in your **gcr.io** registry:
    ```
    gcloud container images list
    ```
    If you are using the gcr.io registry on GCP, the default URI is `gcr.io/<PROJECT_NAME>` - your output will display your gcr.io registry, instead of the example one below:
    ```
    NAME
    gcr.io/matchmaker-dev-201405/openmatch-backendapi
    gcr.io/matchmaker-dev-201405/openmatch-backendclient
    gcr.io/matchmaker-dev-201405/openmatch-clientloadgen
    gcr.io/matchmaker-dev-201405/openmatch-evaluator
    gcr.io/matchmaker-dev-201405/openmatch-frontendapi
    gcr.io/matchmaker-dev-201405/openmatch-frontendclient
    gcr.io/matchmaker-dev-201405/openmatch-mmf
    gcr.io/matchmaker-dev-201405/openmatch-mmforc
    gcr.io/matchmaker-dev-201405/openmatch-mmlogicapi
    ```
* The default example MMF images all use the same name (`openmatch-mmf`), with different image tags designating the different examples.  You can check that these exist by running this command (again, substituting your **gcr.io** registry):
    ```
    gcloud container images list-tags gcr.io/matchmaker-dev-201405/openmatch-mmf
    ```
    You should see tags for several of the example MMFs.  By default, Open Match will try to use the `openmatch-mmf:py3` image in the examples below, so it is important that the image build was successful and a `py3` image tag exists in your **gcr.io** registry before you continue:
    ```
    DIGEST        TAGS     TIMESTAMP
    5345475e026c  php      2018-12-05T00:06:47
    e5c274c3509c  go       2018-12-05T00:02:17
    1b3ec3176d0f  py3      2018-12-05T00:02:07
    ```

## Example of starting a GKE cluster

A cluster with mostly default settings will work for this development guide.  In the Cloud SDK command below we start it with machines that have 4 vCPUs.  Alternatively, you can use the 'Create Cluster' button in [Google Cloud Console]("https://console.cloud.google.com/kubernetes").

```
gcloud container clusters create --machine-type n1-standard-4 open-match-dev-cluster --zone <ZONE>
```

If you don't know which zone to launch the cluster in (`<ZONE>`), you can list all available zones by running the following command.

```
gcloud compute zones list
```

## Configuration

Currently, each component reads a local config file `matchmaker_config.json`, and all components assume they have the same configuration (if you would like to help us design the replacement config solution, please join the [discussion](https://github.com/GoogleCloudPlatform/open-match/issues/42).  To this end, there is a single centralized config file located in the `<REPO_ROOT>/config/` which is symlinked to each component's subdirectory for convenience when building locally.

**NOTE** `defaultImages` container images names in the config file will need to be updated with **your container registry URI**. If you are using the gcr.io registry on GCP, the default URI is `gcr.io/<PROJECT_NAME>`.  Here's an example command in Linux to do this (just replace YOUR_REGISTRY_URI with the appropriate location in your environment, should be run from the config directory):
```
sed -i 's|gcr.io/matchmaker-dev-201405|YOUR_REGISTRY_URI|g' matchmaker_config.json
```
For MacOS the `-i` flag creates backup files when changing the original file in place. You can use the following command, and then delete the `*.backup` files afterwards if you don't need them anymore:
```
sed -i'.backup' -e 's|gcr.io/matchmaker-dev-201405|YOUR_REGISTRY_URI|g' matchmaker_config.json
```

## Running Open Match in a development environment

The rest of this guide assumes you have a cluster (example is using GKE, but works on any cluster with a little tweaking), and kubectl configured to administer that cluster, and you've built all the Docker container images described by `Dockerfiles` in the repository root directory and given them the docker tag 'dev'.  It assumes you are in the `<REPO_ROOT>/deployments/k8s/` directory.

**NOTE** Kubernetes resources that use container images will need to be updated with **your container registry URI**. Here's an example command in Linux to do this (just replace YOUR_REGISTRY_URI with the appropriate location in your environment):
```
sed -i 's|gcr.io/matchmaker-dev-201405|YOUR_REGISTRY_URI|g' *deployment.json
```
For MacOS the `-i` flag creates backup files when changing the original file in place. You can use the following command, and then delete the `*.backup` files afterwards if you don't need them anymore:
```
sed -i'.backup' -e 's|gcr.io/matchmaker-dev-201405|YOUR_REGISTRY_URI|g' *deployment.json
```
If you are using the gcr.io registry on GCP, the default URI is `gcr.io/<PROJECT_NAME>`.

* Start a copy of redis and a service in front of it:
    ```
    kubectl apply -f redis_deployment.json
    kubectl apply -f redis_service.json
    ```
* Run the **core components**: the frontend API, the backend API, the matchmaker function orchestrator (MMFOrc), and the matchmaking logic API.
**NOTE** In order to kick off jobs, the matchmaker function orchestrator needs a service account with permission to administer the cluster. This should be updated to have min required perms before launch, this is pretty permissive but acceptable for closed testing:
    ```
    kubectl apply -f backendapi_deployment.json
    kubectl apply -f backendapi_service.json
    kubectl apply -f frontendapi_deployment.json
    kubectl apply -f frontendapi_service.json
    kubectl apply -f mmforc_deployment.json
    kubectl apply -f mmforc_serviceaccount.json
    kubectl apply -f mmlogic_deployment.json
    kubectl apply -f mmlogic_service.json
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

**Note**: The programs provided below are just bare-bones manual testing programs with no automation and no claim of code coverage. This sparseness of this part of the documentation is because we expect to discard all of these tools and write a fully automated end-to-end test suite and a collection of load testing tools, with extensive stats output and tracing capabilities before 1.0 release. Tracing has to be integrated first, which will be in an upcoming release.

In the end: *caveat emptor*. These tools all work and are quite small, and as such are fairly easy for developers to understand by looking at the code and logging output. They are provided as-is just as a reference point of how to begin experimenting with Open Match integrations.  

* `examples/frontendclient` is a fake client for the Frontend API.  It pretends to be a real game client connecting to Open Match and requests a game, then dumps out the connection string it receives.  Note that it doesn't actually test the return path by looking for arbitrary results from your matchmaking function; it pauses and tells you the name of a key to set a connection string in directly using a redis-cli client.  **Note**: If you're using the rest of these test programs, you're probably using the Backend Client below.  The default profiles that sends to the backend look for way more than one player, so if you want to see meaningful results from running this Frontend Client, you're going to need to generate a bunch of fake players using the client load simulation tool at the same time. Otherwise, expect to wait until it times out as your matchmaker never has enough players to make a successful match.
* `examples/backendclient` is a fake client for the Backend API.  It pretends to be a dedicated game server backend connecting to openmatch and sending in a match profile to fill.  Once it receives a match object with a roster, it will also issue a call to assign the player IDs, and gives an example connection string.  If it never seems to get a match, make sure you're adding players to the pool using the other two tools. Note: building this image requires that you first build the 'base' dev image (look for `cloudbuild_base.yaml` and `Dockerfile.base` in the root directory) and then update the first step to point to that image in your registry.  This will be simplified in a future release.  **Note**: If you run this by itself, expect it to wait about 30 seconds, then return a result of 'insufficient players' and exit - this is working as intended.  Use the client load simulation tool below to add players to the pool or you'll never be able to make a successful match. 
* `test/cmd/client` is a (VERY) basic client load simulation tool.  It does **not** test the Frontend API - in fact, it ignores it and writes players directly to state storage on its own.  It doesn't do anything but loop endlessly, writing players into state storage so you can test your backend integration, and run your custom MMFs and Evaluators (which are only triggered when there are players in the pool). 

### Resources

* [Prometheus Operator spec](https://github.com/coreos/prometheus-operator/blob/master/Documentation/api.md)


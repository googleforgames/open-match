# Development Guide

This doc explains how to setup a development environment, compile from source and deploy your changes to test cluster. This document is targeted to to developers contributing to Open Match.

## Security Disclaimer
**This project has not completed a first-line security audit. This should be fine for testing/development in a local environment, but absolutely should not be used as-is in a production environment.**

## Setting up a local Open Match Repository

Here are the instructions to set up a local repository for Open Match.

```bash
# Install Open Match Toolchain Dependencies (for Debian, other OSes including Mac OS X have similar dependencies)
sudo apt-get update; sudo apt-get install -y -q python3 python3-virtualenv virtualenv make google-cloud-sdk git unzip tar
mkdir -p $HOME/<workspace>
cd $HOME/<workspace>
git clone https://github.com/GoogleCloudPlatform/open-match.git
cd open-match
```

## Compiling From Source

The easiest way to build Open Match is to use the [Makefile](Makefile). This section assumes that you have followed the steps to [Setup Local Open Match Repository](#local-repository-setup).

You will also need [Docker](https://docs.docker.com/install/) and [Go 1.12+](https://golang.org/dl/) installed.

To build all the artifacts of Open Match, please run the following commands:

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

After successfully building, run `docker images` to see all the images that were build.

Before creating a pull request you can run `make local-cloud-build` to simulate a Cloud Build run to check for regressions.

The [Build Queue](https://console.cloud.google.com/cloud-build/builds?project=open-match-build) runs against all PRs, requires membership to [open-match-discuss@googlegroups.com](https://groups.google.com/forum/#!forum/open-match-discuss).

## Deploy Open Match to Google Cloud Platform

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
gcloud compute zones list
```

This section assumes that you have followed the steps to [Setup Local Open Match Repository](#local-repository-setup). Once everything is setup you can deploy Open Match by creating a cluster in Google Kubernetes Engine (GKE).

```bash
# Create a GKE Cluster and install Helm
make create-gke-cluster push-helm
# Push images to Registry
make push-images
# Deploy Open Match
make install-chart
```

This will install all Open Match core components to the kubernetes cluster. Once deployed you can view the jobs in [Cloud Console](https://console.cloud.google.com/kubernetes/workload).

Run `kubectl --namespace open-match get pods,svc` to verify if the deployment succeded. If everything started correctly, the output should look like:

```
$ kubectl --namespace open-match get pods,svc

NAME                                                            READY   STATUS    RESTARTS   AGE
pod/om-backendapi-6f8f9796f7-ncfgf                              1/1     Running   0          10m
pod/om-frontendapi-868f7df859-5dbcd                             1/1     Running   0          10m
pod/om-mmlogicapi-5998dcdc9c-vmjhn                              1/1     Running   0          10m
pod/om-redis-master-0                                           1/1     Running   0          10m
pod/om-redis-metrics-66c8fbfbc-vnmls                            1/1     Running   0          10m
pod/om-redis-slave-8477c666fc-kb2gv                             1/1     Running   1          10m
pod/open-match-grafana-6769f969f-t76zz                          2/2     Running   0          10m
pod/open-match-prometheus-alertmanager-58c9f6ffc7-7f7fq         2/2     Running   0          10m
pod/open-match-prometheus-kube-state-metrics-79c8d85c55-q69qf   1/1     Running   0          10m
pod/open-match-prometheus-node-exporter-88pjh                   1/1     Running   0          10m
pod/open-match-prometheus-node-exporter-qq9h7                   1/1     Running   0          10m
pod/open-match-prometheus-node-exporter-rcmdq                   1/1     Running   0          10m
pod/open-match-prometheus-pushgateway-6c67d47f48-8bhgk          1/1     Running   0          10m
pod/open-match-prometheus-server-86c459ddc4-gk5m7               2/2     Running   0          10m

NAME                                               TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)               AGE
service/om-backendapi                              ClusterIP   10.0.2.206    <none>        50505/TCP,51505/TCP   10m
service/om-frontendapi                             ClusterIP   10.0.14.157   <none>        50504/TCP,51504/TCP   10m
service/om-mmlogicapi                              ClusterIP   10.0.10.71    <none>        50503/TCP,51503/TCP   10m
service/om-redis-master                            ClusterIP   10.0.9.110    <none>        6379/TCP              10m
service/om-redis-metrics                           ClusterIP   10.0.9.114    <none>        9121/TCP              10m
service/om-redis-slave                             ClusterIP   10.0.3.46     <none>        6379/TCP              10m
service/open-match-grafana                         ClusterIP   10.0.0.213    <none>        3000/TCP              10m
service/open-match-prometheus-alertmanager         ClusterIP   10.0.6.126    <none>        80/TCP                10m
service/open-match-prometheus-kube-state-metrics   ClusterIP   None          <none>        80/TCP                10m
service/open-match-prometheus-node-exporter        ClusterIP   None          <none>        9100/TCP              10m
service/open-match-prometheus-pushgateway          ClusterIP   10.0.15.222   <none>        9091/TCP              10m
service/open-match-prometheus-server               ClusterIP   10.0.6.7      <none>        80/TCP                10m
```

## End-to-End testing

### Example MMF, Evaluator

When Open Match is setup, it requires a Match Function and an Evaluator to be set up that it will call into at runtime when requests to generate matches are received. Open Match itself provides harness code (currently only for golang) that abstracts the complexity of setting up the Match Function and Evaluator as GRPC services. You do not need to modify the harness code but simply the actual Match Function, Evaluator Function to suit your game's needs. Open Match includes sample Match function and Evaluation Function as described below:

* `examples/functions/golang/grpc-serving` is a sample Match function that is built using the GRPC harness. The function scans a simple profile, populating a player into each Roster slot that matches the requested player pool. This function is over-simplified simply matching player pools to roster slots. You will need to modify this function to add your match making logic.

* `examples/evaluators/golang/serving` is a sample evaluator function that is called by an evaluator harness that runs as forever-runnig kubernetes job. The function is triggered each time there are results to evaluate. The current sample simply approves matches with unique players, identifies the ones with overlap and approves the first overlapping player match rejecting the rest. You would need to build your own evaluation logic with this sample as a reference.

### Example Tooling

Once Open Match core components are set up and your Match Function and Evaluator GRPC services are running, Open Match functionality is triggered when new players request assignments and when the game backend requests matches. To see Open Match, in action, here are some basic tools that are provided as samples. Note that these tools are meant to exercise Open Match functionality and should only be used as a reference point when building similar abilities into your components using Open Match.

* `test/cmd/clientloadgen/` is a (VERY) basic client load simulation tool.  It does **not** test the Frontend API - in fact, it ignores it and writes players directly to state storage on its own.  It doesn't do anything but loop endlessly, writing players into state storage so you can test your backend integration, and run your custom MMFs and Evaluators (which are only triggered when there are players in the pool).

* `examples/backendclient` is a fake client for the Backend API.  It pretends to be a dedicated game server backend connecting to Open Match and sending in a match profile to fill.  Once it receives a match object with a roster, it will also issue a call to assign the player IDs, and gives an example connection string.  If it never seems to get a match, make sure you're adding players to the pool using the other two tools. **Note**: If you run this by itself, expect it to wait about 30 seconds, then return a result of 'insufficient players' and exit - this is working as intended.  Use the client load simulation tool below to add players to the pool or you'll never be able to make a successful match.

* `test/cmd/frontendclient/` is a fake client for the Frontend API.  It pretends to be group of real game clients connecting to Open Match.  It requests a game, then dumps out the results each player receives to the screen until you press the enter key. **Note**: If you're using the rest of these test programs, you're probably using the Backend Client below.  The default profiles that command sends to the backend look for many more than one player, so if you want to see meaningful results from running this Frontend Client, you're going to need to generate a bunch of fake players using the client load simulation tool at the same time. Otherwise, expect to wait until it times out as your matchmaker never has enough players to make a successful match. Also, if the simulator has generated significant load, the player injected by te Frontend Client may still not find a match by the timeout duration and exit.

### Setting up an E2E scenario

These steps assume that you already have [deployed core Open Match to a cluster](deploy-open-match-to-google-cloud-platform). Once Open Match is deployed, run the below command to deploy the Match Function Harness, the Evaluator and the Client Load Generator to the Open Match cluster.

```bash
# Deploy Open Match
make install-example-chart
```

Once this succeeds, run the below command to validate that these components are up and running as expected (in addition to the Open Match core components):

```bash
$kubectl --namespace open-match get pods,svc

NAME                                                            READY   STATUS    RESTARTS   AGE
pod/om-clientloadgen-b6cf884cd-nslrl                            1/1     Running   0          19s
pod/om-evaluator-7795968f9-729qs                                1/1     Running   0          19s
pod/om-function-697db9cd6-xh89j                                 1/1     Running   0          19s

NAME                                               TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)               AGE
service/om-function                                ClusterIP   10.0.4.81     <none>        50502/TCP,51502/TCP   20s
```

At this point, the Evaluator and Match Function are both running and a client load generator is continuously adding players to the state storage. To see match generation in action, run the following command:

```bash
make run-backendclient
```

Some other handy commands:

```bash
# Cleanup the installation
make delete-chart delete-example-chart

# To install a pre-built image without building again:
make REGISTRY=$REGISTRY TAG=$TAG install-chart install-example-chart
make REGISTRY=$REGISTRY TAG=$TAG install-example-chart
make REGISTRY=$REGISTRY TAG=$TAG run-backendclient
```

**Note**: The programs provided below are just bare-bones manual testing programs with no automation and no claim of code coverage. This sparseness of this part of the documentation is because we expect to discard all of these tools and write a fully automated end-to-end test suite and a collection of load testing tools, with extensive stats output and tracing capabilities before 1.0 release. Tracing has to be integrated first, which will be in an upcoming release.

In the end: *caveat emptor*. These tools all work and are quite small, and as such are fairly easy for developers to understand by looking at the code and logging output. They are provided as-is just as a reference point of how to begin experimenting with Open Match integrations.

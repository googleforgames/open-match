This folder is for the tutorial to deploy the Open Match Matchmaker 102 sample with Redis Enterprise.

## Deploy Redis Enterprise
<TODO - Update the README with stepts to deploy Redis Enterprise>


### Set Redis Port and Password Environment Variables
#### Set Redis Port
```cmd
REDIS-PORT=<Insert Redis instance port>
```

#### Set Redis Password
```cmd
REDIS-PASS=<Replace with Redis Password>
```

#### VPC peering <TODO>
```

```

## Deploy Open Match Core with Redis Enterprise
Run the below command below to deploy Open Match Core with Redis Enterprise via Helm.

```
helm install open-match --create-namespace --namespace open-match open-match/open-match --set open-match-customize.enabled=true --set open-match-customize.evaluator.enabled=true --set open-match-override.enabled=true --set open-match-core.redis.enabled=false --set open-match-core.redis.hostname="Default:$(REDIS-PASS)@redis-$(REDIS-PORT).internal.c22552.us-west1-mz.gcp.cloud.rlrcp.com" --set open-match-core.redis.port=$REDIS-PORT
```

## Deploy Redis Enterprise Tutorial

### Set Environment Variables

#### Set up Image Registry
Please setup an Image registry(such as [Docker Hub](https://hub.docker.com/) or [Google Cloud Container Registry](https://cloud.google.com/container-registry/)) to store the Docker Images used in this tutorial. Once you have set this up, here are the instructions to set up a shell variable that points to your registry:

```cmd
REGISTRY=[YOUR_REGISTRY_URL]
```

If using GKE, you can populate the image registry using the command below:

```cmd
REGISTRY=gcr.io/$(gcloud config list --format 'value(core.project)')
```

#### Get the Tutorial template

Make a local copy of the [Tutorials Folder](https://github.com/googleforgames/open-match/blob/{{< param release_branch >}}/tutorials/matchmaker102). Checkout `release-x.y` branch（e.g. release-1.3, release-1.4） of the Open Match Git repository. Use `tutorials/matchmaker102` of `release-x.y` branch as a working copy for all the instructions in this tutorial.

For convenience, set the following variable:

```cmd
TUTORIALROOT=[SRCROOT]/tutorials/matchmaker102
```

#### Create the Tutorial namespace

Run this command to create a namespace mm102-tutorial in which all the components for this Tutorial will be deployed.

```bash
kubectl create namespace redis-ent
```

### Changes to Components

#### Overview

For this tutorial, we will increase the generation of tickets to show off Redis Enterprise handling ticket generation at scale. Increase this to you liking:

#### Game Frontend

The mock Ticket interation rate is set in `$TUTORIALROOT/frontend/main.go`.

Increase the number of tickets per iteration to any value. For this tutorial we will set it to 7500.

```golang
const (
	// The endpoint for the Open Match Frontend service.
	omFrontendEndpoint = "open-match-frontend.open-match.svc.cluster.local:50504"
	// Number of tickets created per iteration
	ticketsPerIter = 7500
)
```

### Build and Push Images
```
docker build -t $REGISTRY/redis-ent-frontend frontend/
docker push $REGISTRY/redis-ent-frontend
docker build -t $REGISTRY/redis-ent-director director/
docker push $REGISTRY/redis-ent-director
docker build -t $REGISTRY/redis-ent-matchfunction matchfunction/
docker push $REGISTRY/redis-ent-matchfunction
```

### Deploy and Run

Run the below command in the `$TUTORIALROOT` path to deploy the MatchFunction, Game Frontend and the Director to the `redis-ent` namespace:

```cmd
sed "s|REGISTRY_PLACEHOLDER|$REGISTRY|g" matchmaker.yaml | kubectl apply -f -
```

### Output

All the components in this tutorial simply log their progress to stdout. Thus to see the progress, run the below commands:

```bash
kubectl logs -n redis-ent pod/redis-ent-frontend
kubectl logs -n redis-ent pod/redis-ent-director
kubectl logs -n redis-ent pod/redis-ent-matchfunction
```

## Cleanup

Run the command below to remove all the components of this tutorial:

```bash
kubectl delete namespace redis-ent
```

This will delete all the components deployed in this tutorial. Open Match core in open-match namespace can then be reused for other exercises but you will need to re-customize it.
```
# Open Match

Open Match is an open source game matchmaking framework designed to allow game creators to build matchmakers of any size easily and with as much possibility for sharing and code re-use as possible. It’s designed to be flexible (run it anywhere Kubernetes runs), extensible (match logic can be customized to work for any game), and scalable.

Matchmaking is a complicated process, and when large player populations are involved, many popular matchmaking approaches touch on significant areas of computer science including graph theory and massively concurrent processing. Open Match is an effort to provide a foundation upon which these difficult problems can be addressed by the wider game development community. As Josh Menke &mdash; famous for working on matchmaking for many popular triple-A franchises &mdash; put it:

["Matchmaking, a lot of it actually really is just really good engineering. There's a lot of really hard networking and plumbing problems that need to be solved, depending on the size of your audience."](https://youtu.be/-pglxege-gU?t=830)

This project attempts to solve the networking and plumbing problems, so game developers can focus on the logic to match players into great games.

## Disclaimer
This software is currently alpha, and subject to change.  Although Open Match has already been used to run [production workloads within Google](https://cloud.google.com/blog/topics/inside-google-cloud/no-tricks-just-treats-globally-scaling-the-halloween-multiplayer-doodle-with-open-match-on-google-cloud), but it's still early days on the way to our final goal. There's plenty left to write and we welcome contributions. **We strongly encourage you to engage with the community through the [Slack or Mailing lists](#get-involved) if you're considering using Open Match in production before the 1.0 release, as the documentation is likely to lag behind the latest version a bit while we focus on getting out of alpha/beta as soon as possible.**

## Version
[The current stable version in master is 0.2.0 (alpha)](https://github.com/GoogleCloudPlatform/open-match/releases/tag/020).  

# Core Concepts

[Watch the introduction of Open Match at Unite Berlin 2018 on YouTube](https://youtu.be/qasAmy_ko2o)

Open Match is designed to support massively concurrent matchmaking, and to be scalable to player populations of hundreds of millions or more. It attempts to apply stateless web tech microservices patterns to game matchmaking. If you're not sure what that means, that's okay &mdash; it is fully open source and designed to be customizable to fit into your online game architecture &mdash; so have a look a the code and modify it as you see fit.

## Glossary

* **MMF** &mdash; Matchmaking function. This is the customizable matchmaking logic.
* **Component** &mdash; One of the discrete processes in an Open Match deployment. Open Match is composed of multiple scalable microservices called 'components'.
* **Roster** &mdash; A list of all the players in a match.
* **Profile** &mdash; The json blob containing all the parameters used to select which players go into a roster.
* **Match Object** &mdash; A protobuffer message format that contains the Profile and the results of the matchmaking function. Sent to the backend API from your game backend with an empty roster  and then returned from your MMF with the matchmaking results filled in.
* **MMFOrc** &mdash; Matchmaker function orchestrator. This Open Match core component is in charge of kicking off custom matchmaking functions (MMFs) and evaluator processes.
* **State Storage** &mdash; The storage software used by Open Match to hold all the matchmaking state. Open Match ships with [Redis](https://redis.io/) as the default state storage.
* **Assignment** &mdash; Refers to assigning a player or group of players to a dedicated game server instance. Open Match offers a path to send dedicated game server connection details from your backend to your game clients after a match has been made.
* **DGS** &mdash; Dedicated game server

## Requirements
* [Kubernetes](https://kubernetes.io/) cluster &mdash; tested with version 1.9.
* [Redis 4+](https://redis.io/) &mdash; tested with 4.0.11.
* Open Match is compiled against the latest release of [Golang](https://golang.org/) &mdash; tested with 1.10.3.

## Components

Open Match is a set of processes designed to run on Kubernetes. It contains these **core** components:

1. Frontend API
1. Backend API
1. Matchmaker Function Orchestrator (MMFOrc)
1. Matchmaking Logic (MMLogic) API 

It also explicitly depends on these two **customizable** components.

1. Matchmaking "Function" (MMF)
1. Evaluator (may be deprecated in future versions)

While **core** components are fully open source and *can* be modified, they are designed to support the majority of matchmaking scenarios *without need to change the source code*. The Open Match repository ships with simple **customizable** example MMF and Evaluator processes, but it is expected that most users will want full control over the logic in these, so they have been designed to be as easy to modify or replace as possible.

### Frontend API

The Frontend API accepts the player data and puts it in state storage so your Matchmaking Function (MMF) can access it.

The Frontend API is a server application that implements the [gRPC](https://grpc.io/) service defined in `api/protobuf-spec/frontend.proto`. At the most basic level, it expects clients to connect and send:
* A **unique ID** for the group of players (the group can contain any number of players, including only one).
* A **json blob** containing all player-related data you want to use in your matchmaking function.

The client is expected to maintain a connection, waiting for an update from the API that contains the details required to connect to a dedicated game server instance (an 'assignment'). There are also basic functions for removing an ID from the matchmaking pool or an existing match.

### Backend API

The Backend API puts match profiles in state storage which the Matchmaking Function (MMF) can access and use to decide which players should be put into a match together, then return those matches to dedicated game server instances.

The Backend API is a server application that implements the [gRPC](https://grpc.io/) service defined in `api/protobuf-spec/backend.proto`. At the most basic level, it expects to be connected to your online infrastructure (probably to your server scaling manager or scheduler, or even directly to a dedicated game server), and to receive:
* A **unique ID** for a matchmaking profile.
* A **json blob** containing all the match-related data you want to use in your matchmaking function, in an 'empty' match object.

Your game backend is expected to maintain a connection, waiting for 'filled' match objects containing a roster of players. The Backend API also provides a return path for your game backend to return dedicated game server connection details (an 'assignment') to the game client, and to delete these 'assignments'.


### Matchmaking Function Orchestrator (MMFOrc)

The MMFOrc kicks off your custom matchmaking function (MMF) for every profile submitted to the Backend API. It also runs the Evaluator to resolve conflicts in case more than one of your profiles matched the same players.

The MMFOrc exists to orchestrate/schedule your **custom components**, running them as often as required to meet the demands of your game. MMFOrc runs in an endless loop, submitting MMFs and Evaluator jobs to Kubernetes.

### Matchmaking Logic (MMLogic) API

The MMLogic API provides a series of gRPC functions that act as a Matchmaking Function SDK.  Much of the basic, boilerplate code for an MMF is the same regardless of what players you want to match together.  The MMLogic API offers a gRPC interface for many common MMF tasks, such as:

1. Reading a profile from state storage.
1. Running filters on players in state strorage.
1. Removing chosen players from consideration by other MMFs (by adding them to an ignore list).
1. Writing the matchmaking results to state storage.
1. (Optional, NYI) Exporting MMF stats for metrics collection.

More details about the available gRPC calls can be found in the [API Specification](api/protobuf-spec/messages.proto).

**Note**: using the MMLogic API is **optional**.  It tries to simplify the development of MMFs, but if you want to take care of these tasks on your own, you can make few or no calls to the MMLogic API as long as your MMF still completes all the required tasks.  Read the [Matchmaking Functions section](#matchmaking-functions-mmfs) for more details of what work an MMF must do.

### Evaluator

The Evaluator resolves conflicts when multiple matches want to include the same player(s).

The Evaluator is a component run by the Matchmaker Function Orchestrator (MMFOrc) after the matchmaker functions have been run, and some proposed results are available.  The Evaluator looks at all the proposed matches, and if multiple proposals contain the same player(s), it breaks the tie. In many simple matchmaking setups with only a few game modes and matchmaking functions that always look at different parts of the matchmaking pool, the Evaluator may functionally be a no-op or first-in-first-out algorithm. In complex matchmaking setups where, for example, a player can queue for multiple types of matches, the Evaluator provides the critical customizability to evaluate all available proposals and approve those that will passed to your game servers.

Large-scale concurrent matchmaking functions is a complex topic, and users who wish to do this are encouraged to engage with the [Open Match community](https://github.com/GoogleCloudPlatform/open-match#get-involved) about patterns and best practices.

### Matchmaking Functions (MMFs)

Matchmaking Functions (MMFs) are run by the Matchmaker Function Orchestrator (MMFOrc) &mdash; once per profile it sees in state storage. The MMF is run as a Job in Kubernetes, and has full access to read and write from state storage. At a high level, the encouraged pattern is to write a MMF in whatever language you are comfortable in that can do the following things:

- [x] Be packaged in a (Linux) Docker container.
- [x] Read/write from the Open Match state storage &mdash; Open Match ships with Redis as the default state storage.
- [x] Read a profile you wrote to state storage using the Backend API.
- [x] Select from the player data you wrote to state storage using the Frontend API.
- [ ] Run your custom logic to try to find a match.
- [x] Write the match object it creates to state storage at a specified key.
- [x] Remove the players it selected from consideration by other MMFs.
- [x] Notify the MMFOrc of completion.
- [x] (Optional, but recommended) Export stats for metrics collection.

**Open Match offers [matchmaking logic API](#matchmaking-logic-mmlogic-api) calls for handling the checked items, as long as you are able to format your input and output in the data schema Open Match expects (defined in the [protobuf messages](api/protobuf-spec/messages.proto)).**  You can to do this work yourself if you don't want to or can't use the data schema Open Match is looking for.  However, the data formats expected by Open Match are pretty generalized and will work with most common matchmaking scenarios and game types.  If you have questions about how to fit your data into the formats specified, feel free to ask us in the [Slack or mailing group](#get-involved).

Example MMFs are provided in these languages:
- [C#](examples/functions/csharp/simple) (doesn't use the MMLogic API)
- [Python3](examples/functions/python3/mmlogic-simple) (MMLogic API enabled)
- [PHP](examples/functions/php/mmlogic-simple)  (MMLogic API enabled)
- [golang](examples/functions/golang/manual-simple)  (doesn't use the MMLogic API)

## Open Source Software integrations

### Structured logging

Logging for Open Match uses the [Golang logrus module](https://github.com/sirupsen/logrus) to provide structured logs. Logs are output to `stdout` in each component, as expected by Docker and Kubernetes. Level and format are configurable via config/matchmaker_config.json. If you have a specific log aggregator as your final destination, we recommend you have a look at the logrus documentation as there is probably a log formatter that plays nicely with your stack.

### Instrumentation for metrics

Open Match uses [OpenCensus](https://opencensus.io/) for metrics instrumentation. The [gRPC](https://grpc.io/) integrations are built-in, and Golang redigo module integrations are incoming, but [haven't been merged into the official repo](https://github.com/opencensus-integrations/redigo/pull/1). All of the core components expose HTTP `/metrics` endpoints on the port defined in `config/matchmaker_config.json` (default: 9555) for Prometheus to scrape. If you would like to export to a different metrics aggregation platform, we suggest you have a look at the OpenCensus documentation &mdash; there may be one written for you already, and switching to it may be as simple as changing a few lines of code.

**Note:** A standard for instrumentation of MMFs is planned.

### Redis setup

By default, Open Match expects you to run Redis *somewhere*. Connection information can be put in the config file (`matchmaker_config.json`) for any Redis instance reachable from the [Kubernetes namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/). By default, Open Match sensibly runs in the Kubernetes `default` namespace. In most instances, we expect users will run a copy of Redis in a pod in Kubernetes, with a service pointing to it.

* HA configurations for Redis aren't implemented by the provided Kubernetes resource definition files, but Open Match expects the Redis service to be named `redis`, which provides an easier path to multi-instance deployments.

## Additional examples

**Note:** These examples will be expanded on in future releases.

The following examples of how to call the APIs are provided in the repository. Both have a `Dockerfile` and `cloudbuild.yaml` files in their respective directories:

* `test/cmd/frontendclient/main.go` acts as a client to the the Frontend API, putting a player into the queue with simulated latencies from major metropolitan cities and a couple of other matchmaking attributes. It then waits for you to manually put a value in Redis to simulate a server connection string being written using the backend API 'CreateAssignments' call, and displays that value on stdout for you to verify.
* `examples/backendclient/main.go` calls the Backend API and passes in the profile found in `backendstub/profiles/testprofile.json` to the `ListMatches` API endpoint, then continually prints the results until you exit, or there are insufficient players to make a match based on the profile..

## Usage

Documentation and usage guides on how to set up and customize Open Match.

### Precompiled container images

Once we reach a 1.0 release, we plan to produce publicly available (Linux) Docker container images of major releases in a public image registry. Until then, refer to the 'Compiling from source' section below.

### Compiling from source

All components of Open Match produce (Linux) Docker container images as artifacts, and there are included `Dockerfile`s for each. [Google Cloud Platform Cloud Build](https://cloud.google.com/cloud-build/docs/) users will also find `cloudbuild.yaml` files for each component in the corresponding `cmd/<COMPONENT>` directories.

All the core components for Open Match are written in Golang and use the [Dockerfile multistage builder pattern](https://docs.docker.com/develop/develop-images/multistage-build/). This pattern uses intermediate Docker containers as a Golang build environment while producing lightweight, minimized container images as final build artifacts. When the project is ready for production, we will modify the `Dockerfile`s to uncomment the last build stage. Although this pattern is great for production container images, it removes most of the utilities required to troubleshoot issues during development.

## Configuration
Currently, each component reads a local config file `matchmaker_config.json`, and all components assume they have the same configuration. To this end, there is a single centralized config file located in the `<REPO_ROOT>/config/` which is symlinked to each component's subdirectory for convenience when building locally. When `docker build`ing the component container images, the Dockerfile copies the centralized config file into the component directory.

We plan to replace this with a Kubernetes-managed config with dynamic reloading when development time allows. Pull requests are welcome!

### Guides
* [Production guide](./docs/production.md) Lots of best practices to be written here before 1.0 release. **WIP**
* [Development guide](./docs/development.md)

### Reference
* [FAQ](./docs/faq.md)

## Get involved

* [Slack channel](https://open-match.slack.com/)
    * [Signup link](https://join.slack.com/t/open-match/shared_invite/enQtNDM1NjcxNTY4MTgzLWQzMzE1MGY5YmYyYWY3ZjE2MjNjZTdmYmQ1ZTQzMmNiNGViYmQyN2M4ZmVkMDY2YzZlOTUwMTYwMzI1Y2I2MjU)
* [Mailing list](https://groups.google.com/forum/#!forum/open-match-discuss)
* [Managed Service Survey](https://goo.gl/forms/cbrFTNCmy9rItSv72)

## Code of Conduct

Participation in this project comes under the [Contributor Covenant Code of Conduct](code-of-conduct.md)

## Development and Contribution

Please read the [contributing](CONTRIBUTING.md) guide for directions on submitting Pull Requests to Open Match.

See the [Development guide](docs/development.md) for documentation for development and building Open Match from source.

The [Release Process](docs/governance/release_process.md) documentation displays the project's upcoming release calendar and release process. (NYI)

Open Match is in active development - we would love your help in shaping its future!

## This all sounds great, but can you explain Docker and/or Kubernetes to me?

### Docker
- [Docker's official "Getting Started" guide](https://docs.docker.com/get-started/)
- [Katacoda's free, interactive Docker course](https://www.katacoda.com/courses/docker)

### Kubernetes
- [You should totally read this comic, and interactive tutorial](https://cloud.google.com/kubernetes-engine/kubernetes-comic/)
- [Katacoda's free, interactive Kubernetes course](https://www.katacoda.com/courses/kubernetes)

## Licence

Apache 2.0


# Planned improvements

## Documentation 
- [ ] “Writing your first matchmaker” getting started guide will be included in an upcoming version.
- [ ] Documentation for using the example customizable components and the `backendstub` and `frontendstub` applications to do an end-to-end (e2e) test will be written. This all works now, but needs to be written up.
- [ ] Documentation on release process and release calendar.

## State storage
- [ ] All state storage operations should be isolated from core components into the `statestorage/` modules.  This is necessary precursor work to enabling Open Match state storage to use software other than Redis.
- [ ] [The Redis deployment should have an example HA configuration](https://github.com/GoogleCloudPlatform/open-match/issues/41)
- [ ] Redis watch should be unified to watch a hash and stream updates.  The code for this is written and validated but not committed yet. We don't want to support two redis watcher code paths, so the backend watch of the match object should be switched to unify the way the frontend and backend watch keys.  The backend part of this is in but the frontend part is in another branch and will be committed later. 
- [ ] Player/Group records generated when a client enters the matchmaking pool need to be removed after a certain amount of time with no activity. When using Redis, this will be implemented as a expiration on the player record.

## Instrumentation / Metrics / Analytics
- [ ] Instrumentation of MMFs is in the planning stages.  Since MMFs are by design meant to be completely customizable (to the point of allowing any process that can be packaged in a Docker container), metrics/stats will need to have an expected format and formalized outgoing pathway.  Currently the thought is that it might be that the metrics should be written to a particular key in statestorage in a format compatible with opencensus, and will be collected, aggreggated, and exported to Prometheus using another process.
- [ ] [OpenCensus tracing](https://opencensus.io/core-concepts/tracing/) will be implemented in an upcoming version.
- [ ] Read logrus logging configuration from matchmaker_config.json.

## Security
- [ ] The Kubernetes service account used by the MMFOrc should be updated to have min required permissions.

## Kubernetes
- [ ] Autoscaling isn't turned on for the Frontend or Backend API Kubernetes deployments by default.
- [ ] A [Helm](https://helm.sh/) chart to stand up Open Match will be provided in an upcoming version. For now just use the [installation YAMLs](./install/yaml).

## CI / CD / Build
- [ ] We plan to host 'official' docker images for all release versions of the core components in publicly available docker registries soon.
- [ ] CI/CD for this repo and the associated status tags are planned.
- [ ] Golang unit tests will be shipped in an upcoming version.
- [ ] A full load-testing and e2e testing suite will be included in an upcoming version.

## Will not Implement 
- [X] Defining multiple images inside a profile for the purposes of experimentation adds another layer of complexity into profiles that can instead be handled outside of open match with custom match functions in collaboration with a director (thing that calls backend to schedule matchmaking) 

### Special Thanks
- Thanks to https://jbt.github.io/markdown-editor/ for help in marking this document down.

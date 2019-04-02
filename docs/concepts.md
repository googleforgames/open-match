
# Core Concepts

[Watch the introduction of Open Match at Unite Berlin 2018 on YouTube](https://youtu.be/qasAmy_ko2o)

Open Match is designed to support massively concurrent matchmaking, and to be scalable to player populations of hundreds of millions or more. It attempts to apply stateless web tech microservices patterns to game matchmaking. If you're not sure what that means, that's okay &mdash; it is fully open source and designed to be customizable to fit into your online game architecture &mdash; so have a look a the code and modify it as you see fit.

## Glossary

### General
* **DGS** &mdash; Dedicated game server
* **Client** &mdash; The game client program the player uses when playing the game
* **Session** &mdash; In Open Match, players are matched together, then assigned to a server which hosts the game _session_.  Depending on context, this may be referred to as a _match_, _map_, or just _game_ elsewhere in the industry.  

### Open Match
* **Component** &mdash; One of the discrete processes in an Open Match deployment. Open Match is composed of multiple scalable microservices called _components_.
* **State Storage** &mdash; The storage software used by Open Match to hold all the matchmaking state. Open Match ships with [Redis](https://redis.io/) as the default state storage.
* **MMFOrc** &mdash; Matchmaker function orchestrator. This Open Match core component is in charge of kicking off custom matchmaking functions (MMFs) and evaluator processes.
* **MMF** &mdash; Matchmaking function. This is the customizable matchmaking logic.
* **MMLogic API** &mdash; An API that provides MMF SDK functionality. It is optional - you can also do all the state storage read and write operations yourself if you have a good reason to do so.
* **Director** &mdash; The software you (as a developer) write against the Open Match Backend API. The _Director_ decides which MMFs to run, and is responsible for sending MMF results to a DGS to host the session.

### Data Model 
* **Player** &mdash; An ID and list of attributes with values for a player who wants to participate in matchmaking.
* **Roster** &mdash; A list of player objects.  Used to hold all the players on a single team.
* **Filter** &mdash; A _filter_ is used to narrow down the players to only those who have an attribute value within a certain integer range.  All attributes are integer values in Open Match because [that is how indices are implemented](internal/statestorage/redis/playerindices/playerindices.go). A _filter_ is defined in a _player pool_.
* **Player Pool** &mdash; A list of all the players who fit all the _filters_ defined in the pool.
* **Match Object** &mdash; A protobuffer message format that contains the _profile_ and the results of the matchmaking function. Sent to the backend API from your game backend with the _roster_(s) empty and then returned from your MMF with the matchmaking results filled in.
* **Profile** &mdash; The json blob containing all the parameters used by your MMF to select which players go into a roster together.
* **Assignment** &mdash; Refers to assigning a player or group of players to a dedicated game server instance. Open Match offers a path to send dedicated game server connection details from your backend to your game clients after a match has been made.
* **Ignore List** &mdash; Removing players from matchmaking consideration is accomplished using _ignore lists_.  They contain lists of player IDs that your MMF should not include when making matches.

## Requirements
* [Kubernetes](https://kubernetes.io/) cluster &mdash; tested with version 1.11.7.
* [Redis 4+](https://redis.io/) &mdash; tested with 4.0.11.
* Open Match is compiled against the latest release of [Golang](https://golang.org/) &mdash; tested with 1.11.5.

## Components

Open Match is a set of processes designed to run on Kubernetes. It contains these **core** components:

1. Frontend API
1. Backend API
1. Matchmaker Function Orchestrator (MMFOrc) (may be deprecated in future versions)

It includes these **optional** (but recommended) components:
1. Matchmaking Logic (MMLogic) API 

It also explicitly depends on these two **customizable** components.

1. Matchmaking "Function" (MMF)
1. Evaluator (may be optional in future versions)

While **core** components are fully open source and _can_ be modified, they are designed to support the majority of matchmaking scenarios *without need to change the source code*. The Open Match repository ships with simple **customizable** MMF and Evaluator examples, but it is expected that most users will want full control over the logic in these, so they have been designed to be as easy to modify or replace as possible.

### Frontend API

The Frontend API accepts the player data and puts it in state storage so your Matchmaking Function (MMF) can access it.

The Frontend API is a server application that implements the [gRPC](https://grpc.io/) service defined in `api/protobuf-spec/frontend.proto`. At the most basic level, it expects clients to connect and send:
* A **unique ID** for the group of players (the group can contain any number of players, including only one).
* A **json blob** containing all player-related data you want to use in your matchmaking function.

The client is expected to maintain a connection, waiting for an update from the API that contains the details required to connect to a dedicated game server instance (an 'assignment'). There are also basic functions for removing an ID from the matchmaking pool or an existing match.

### Backend API

The Backend API writes match objects to state storage which the Matchmaking Functions (MMFs) access to decide which players should be matched. It returns the results from those MMFs.

The Backend API is a server application that implements the [gRPC](https://grpc.io/) service defined in `api/protobuf-spec/backend.proto`. At the most basic level, it expects to be connected to your online infrastructure (probably to your server scaling manager or **director**, or even directly to a dedicated game server), and to receive:
* A **unique ID** for a matchmaking profile.
* A **json blob** containing all the matching-related data and filters you want to use in your matchmaking function.
* An optional list of **roster**s to hold the resulting teams chosen by your matchmaking function.
* An optional set of **filters** that define player pools your matchmaking function will choose players from.  

Your game backend is expected to maintain a connection, waiting for 'filled' match objects containing a roster of players. The Backend API also provides a return path for your game backend to return dedicated game server connection details (an 'assignment') to the game client, and to delete these 'assignments'.  


### Matchmaking Function Orchestrator (MMFOrc)

The MMFOrc kicks off your custom matchmaking function (MMF) for every unique profile submitted to the Backend API in a match object. It also runs the Evaluator to resolve conflicts in case more than one of your profiles matched the same players.

The MMFOrc exists to orchestrate/schedule your **custom components**, running them as often as required to meet the demands of your game. MMFOrc runs in an endless loop, submitting MMFs and Evaluator jobs to Kubernetes.

### Matchmaking Logic (MMLogic) API

The MMLogic API provides a series of gRPC functions that act as a Matchmaking Function SDK.  Much of the basic, boilerplate code for an MMF is the same regardless of what players you want to match together.  The MMLogic API offers a gRPC interface for many common MMF tasks, such as:

1. Reading a profile from state storage.
1. Running filters on players in state strorage. It automatically removes players on ignore lists as well!
1. Removing chosen players from consideration by other MMFs (by adding them to an ignore list). It does it automatically for you when writing your results!
1. Writing the matchmaking results to state storage.
1. (Optional, NYI) Exporting MMF stats for metrics collection.

More details about the available gRPC calls can be found in the [API Specification](api/protobuf-spec/messages.proto).

**Note**: using the MMLogic API is **optional**.  It tries to simplify the development of MMFs, but if you want to take care of these tasks on your own, you can make few or no calls to the MMLogic API as long as your MMF still completes all the required tasks.  Read the [Matchmaking Functions section](#matchmaking-functions-mmfs) for more details of what work an MMF must do.

### Evaluator

The Evaluator resolves conflicts when multiple MMFs select the same player(s).

The Evaluator is a component run by the Matchmaker Function Orchestrator (MMFOrc) after the matchmaker functions have been run, and some proposed results are available.  The Evaluator looks at all the proposals, and if multiple proposals contain the same player(s), it breaks the tie. In many simple matchmaking setups with only a few game modes and well-tuned matchmaking functions, the Evaluator may functionally be a no-op or first-in-first-out algorithm. In complex matchmaking setups where, for example, a player can queue for multiple types of matches, the Evaluator provides the critical customizability to evaluate all available proposals and approve those that will passed to your game servers.

Large-scale concurrent matchmaking functions is a complex topic, and users who wish to do this are encouraged to engage with the [Open Match community](https://github.com/GoogleCloudPlatform/open-match#get-involved) about patterns and best practices.

### Matchmaking Functions (MMFs)

Matchmaking Functions (MMFs) are run by the Matchmaker Function Orchestrator (MMFOrc) &mdash; once per profile it sees in state storage. The MMF is run as a Job in Kubernetes, and has full access to read and write from state storage. At a high level, the encouraged pattern is to write a MMF in whatever language you are comfortable in that can do the following things:

- [x] Be packaged in a (Linux) Docker container.
- [x] Read/write from the Open Match state storage &mdash; Open Match ships with Redis as the default state storage.
- [x] Read a profile you wrote to state storage using the Backend API.
- [x] Select from the player data you wrote to state storage using the Frontend API.  It must respect all the ignore lists defined in the matchmaker config.
- [ ] Run your custom logic to try to find a match.
- [x] Write the match object it creates to state storage at a specified key.
- [x] Remove the players it selected from consideration by other MMFs by adding them to the appropriate ignore list.
- [x] Notify the MMFOrc of completion.
- [x] (Optional, but recommended) Export stats for metrics collection.

**Open Match offers [matchmaking logic API](#matchmaking-logic-mmlogic-api) calls for handling the checked items, as long as you are able to format your input and output in the data schema Open Match expects (defined in the [protobuf messages](api/protobuf-spec/messages.proto)).**  You can to do this work yourself if you don't want to or can't use the data schema Open Match is looking for.  However, the data formats expected by Open Match are pretty generalized and will work with most common matchmaking scenarios and game types.  If you have questions about how to fit your data into the formats specified, feel free to ask us in the [Slack or mailing group](#get-involved).

Example MMFs are provided in these languages:
- [C#](examples/functions/csharp/simple) (doesn't use the MMLogic API)
- [Python3](examples/functions/python3/mmlogic-simple) (MMLogic API enabled)
- [PHP](examples/functions/php/mmlogic-simple)  (MMLogic API enabled)
- [golang](examples/functions/golang/manual-simple)  (doesn't use the MMLogic API)

## Additional examples

**Note:** These examples will be expanded on in future releases.

The following examples of how to call the APIs are provided in the repository. Both have a `Dockerfile` and `cloudbuild.yaml` files in their respective directories:

* `test/cmd/frontendclient/main.go` acts as a client to the the Frontend API, putting a player into the queue with simulated latencies from major metropolitan cities and a couple of other matchmaking attributes. It then waits for you to manually put a value in Redis to simulate a server connection string being written using the backend API 'CreateAssignments' call, and displays that value on stdout for you to verify.
* `examples/backendclient/main.go` calls the Backend API and passes in the profile found in `backendstub/profiles/testprofile.json` to the `ListMatches` API endpoint, then continually prints the results until you exit, or there are insufficient players to make a match based on the profile..

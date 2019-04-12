---
title: "Concepts"
linkTitle: "Concepts"
weight: 4
description: >
  A short lead descripton about this section page. Text here can also be **bold** or _italic_ and can even be split over multiple paragraphs.
---

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
* **Filter** &mdash; A _filter_ is used to narrow down the players to only those who have an attribute value within a certain integer range.  All attributes are integer values in Open Match because that is how indices are implemented. A _filter_ is defined in a _player pool_.
* **Player Pool** &mdash; A list of all the players who fit all the _filters_ defined in the pool.
* **Match Object** &mdash; A protobuffer message format that contains the _profile_ and the results of the matchmaking function. Sent to the backend API from your game backend with the _roster_(s) empty and then returned from your MMF with the matchmaking results filled in.
* **Profile** &mdash; The json blob containing all the parameters used by your MMF to select which players go into a roster together.
* **Assignment** &mdash; Refers to assigning a player or group of players to a dedicated game server instance. Open Match offers a path to send dedicated game server connection details from your backend to your game clients after a match has been made.
* **Ignore List** &mdash; Removing players from matchmaking consideration is accomplished using _ignore lists_.  They contain lists of player IDs that your MMF should not include when making matches.

## Requirements
* [Kubernetes](https://kubernetes.io/) cluster &mdash; tested with version 1.11.7.
* [Redis 4+](https://redis.io/) &mdash; tested with 4.0.11.
* Open Match is compiled against the latest release of [Golang](https://golang.org/) &mdash; tested with 1.11.5.

## Additional examples

**Note:** These examples will be expanded on in future releases.

The following examples of how to call the APIs are provided in the repository. Both have a `Dockerfile` and `cloudbuild.yaml` files in their respective directories:

* `test/cmd/frontendclient/main.go` acts as a client to the the Frontend API, putting a player into the queue with simulated latencies from major metropolitan cities and a couple of other matchmaking attributes. It then waits for you to manually put a value in Redis to simulate a server connection string being written using the backend API 'CreateAssignments' call, and displays that value on stdout for you to verify.
* `examples/backendclient/main.go` calls the Backend API and passes in the profile found in `backendstub/profiles/testprofile.json` to the `ListMatches` API endpoint, then continually prints the results until you exit, or there are insufficient players to make a match based on the profile..

## Licence

Apache 2.0

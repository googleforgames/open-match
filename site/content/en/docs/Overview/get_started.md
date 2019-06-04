---
title: "Get Started Guide"
linkTitle: "Get Started"
weight: 1
description: Follow these steps to get familiar with using Open Match.
---
# Get Started
- [Concept](#concepts)
- [Match Making Lifecycle](#match-making-lifecycle)

[TODO]: # (- [Generate your first match](#generate-your-first-match))
[TODO]: # (- [Customize your match function](#customize-your-match-function))


## Concepts
- _Tickets_: a basic matchmaking entity in Open Match that could be used to represent an individual  or players in a group.
- _Assignment_: an object represents the game server that a ticket binds with.
- _Filter_: an object used to query tickets that meets certain filtering criteria.
- _Pool_: an object consists of different filters and an unique ID.
- _Roster_: a named collection of ticket IDs. It is used to represent a team/substeam in a match.
- _MatchProfile_: an object that defines the shape of a match.
- _Match_: an object abstraction of an actual match.
- _Frontend_: a service that manages tickets in open-match.
- _Backend_: a service that generats matches and host assignments in open-match.
- _Mmlogic_: a gateway service that supports data querying in open-match.
- _Matchfunction_: a service where your custom match making logic lives in.

See deatailed explanation in [Core Concepts](https://github.com/googleforgames/open-match/blob/master/docs/concepts.md) and our [Protobuf Definitions](https://github.com/googleforgames/open-match/blob/master/api/messages.proto).


## Match Making Lifecycle

[TODO]: # (add a chart to illustrate dataflow in open-match)
[TODO]: # (the chart is not added because we have not finalize the API changes yet.)

[TODO]: # (`Generate your first match` and `Customize your match function` are intentionally left blank as the current user experience is pretty bad if you simply wanna try out your customize match function but later find out you have to configure your gcr registry, gcloud account, and wait for 10 minutes to rebuild everything from scratch. We might need to bring skaffold to open-match for the community developers?)


1. A service (potentially a lobby service) figures out a match-making entity (aka. _ticket_) is in-queue for a game match, it then gets a frontend client and triggers the `CreateTicket` function of the frontend service.
2. The frontend service receives the request, indexes the attributes of the entity with its storage service and acknowledges this request.
3. Later at some point, a service that wants a match (aka. _director service_) triggers the `FetchMatches` function of the backend service.
4. The backend service receives the request, and requests a `Run` from the matchfunction service.
5. The matchfunction service first gets data it needs from `QueryTickets` function of the mmlogic service with the storage service, then run its customized match making logic against the response to create match candidates.
6. The backend service receives a match from the matchfunction service, and responds to the director with the match.
7. Director receives the response from the backend service, makes a host assignment to the match, and triggers `AssignTickets` function of the backend service to update ticket status in the storage service.

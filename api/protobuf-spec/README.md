## REST compatibility
Follow the guidelines at https://cloud.google.com/endpoints/docs/grpc/transcoding to keep the gRPC service definitions friendly to REST transcoding. An excerpt:

"Transcoding involves mapping HTTP/JSON requests and their parameters to gRPC methods and their parameters and return types (we'll look at exactly how you do this in the following sections). Because of this, while it's possible to map an HTTP/JSON request to any arbitrary API method, it's simplest and most intuitive to do so if the gRPC API itself is structured in a resource-oriented way, just like a traditional HTTP REST API. In other words, the API service should be designed so that it uses a small number of standard methods (corresponding to HTTP verbs like GET, PUT, and so on) that operate on the service's resources (and collections of resources, which are themselves a type of resource). These standard methods are List, Get, Create, Update, and Delete."

It is for these reasons we don't have gRPC calls that support bi-directional streaming in Open Match.

## REST API Usage
Open Match gateway proxy transcodes any REST calls to its underlying gRPC service. Follow the [examples](https://cloud.google.com/endpoints/docs/grpc-service-config/reference/rpc/google.api#httprule) for further details.

A typical REST call to Open Match backend's `CreateAssignments` service via HTTP POST request 
```
/v1/backend/assignments/123? \
    assignment.rosters.name=foo&assignment.rosters.players.id=1&assignment.rosters.players.id=2
```
is equivalent to 

```go
CreateAssignmentsRequest(
    Assignments(
        name: '123',
        rosters: [
            Roster(name: 'foo', [Player(id: 1), Player(id: 2)])
        ]
    )
)
```

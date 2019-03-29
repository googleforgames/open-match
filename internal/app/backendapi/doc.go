/*
BackendAPI contains the unique files required to run the API endpoints for
Open Match's backend. It is assumed you'll either integrate calls to these
endpoints directly into your dedicated game server (simple use case), or call
these endpoints from other, established services in your infrastructure (more
complicated use cases).

Note that the main package for backendapi does very little except read the
config and set up logging and metrics, then start the server.  Almost all the
work is being done by backendapi/apisrv, which implements the gRPC server
defined in the backendapi/proto/backend.pb.go file.
*/

package backendapi

/*
FrontendAPI contains the unique files required to run the API endpoints for
Open Match's frontend. It is assumed you'll either integrate calls to these
endpoints directly into your game client (simple use case), or call these
endpoints from other, established platform services in your infrastructure
(more complicated use cases).

Note that the main package for frontendapi does very little except read the
config and set up logging and metrics, then start the server.  Almost all the
work is being done by frontendapi/apisrv, which implements the gRPC server
defined in the frontendapi/proto/frontend.pb.go file.

<TODO - UPDATE THIS>
*/

package main

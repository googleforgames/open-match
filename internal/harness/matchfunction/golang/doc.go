/*
MatchFunction harness contains the files required to run the harness as a GRPC
service. To use this harness, you should author the match making function and
pass that in as the callback when setting up the function harness service.

Note that the main package for the harness does very little except read the
config and set up logging and metrics, then start the server. The harness
functionality is implemented in harness/apisrv, which implements the gRPC server
defined in the frontendapi/proto/matchfunction.pb.go file.
*/

package harness

// Package main defines a sample match function that uses the GRPC harness to set up
// the match making function as a service. This sample is a reference
// to demonstrate the usage of the GRPC harness and should only be used as
// a starting point for your match function. You will need to modify the
// matchmaking logic in this function based on your game's requirements.
package main

import (
	"open-match.dev/open-match/examples/functions/golang/backfill/mmf"
)

const (
	queryServiceAddr = "open-match-query.open-match.svc.cluster.local:50503" // Address of the QueryService endpoint.
	serverPort       = 50502                                                 // The port for hosting the Match Function.
)

func main() {
	mmf.Start(queryServiceAddr, serverPort)
}

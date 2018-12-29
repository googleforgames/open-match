package main

import (
	"github.com/GoogleCloudPlatform/open-match/internal/pb"
)

// Allocator is the interface that allocates a game server for a match object
type Allocator interface {
	Allocate(*pb.MatchObject) (string, error)
	Send(string, *pb.MatchObject) error
	UnAllocate(string) error
}

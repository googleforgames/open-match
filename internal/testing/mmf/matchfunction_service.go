// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package mmf provides the Match Making Function service for Open Match golang harness.
package mmf

import (
	"context"

	"golang.org/x/sync/errgroup"
	"open-match.dev/open-match/pkg/pb"
)

// MatchFunction is the function signature for the Match Making Function (MMF) to be implemented by the user.
type MatchFunction func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error

type matchFunctionService struct {
	mmf MatchFunction
}

func (s *matchFunctionService) Run(req *pb.RunRequest, stream pb.MatchFunction_RunServer) error {
	g, ctx := errgroup.WithContext(stream.Context())

	out := make(chan *pb.Match)

	g.Go(func() error {
		defer close(out)
		return s.mmf(ctx, req.Profile, out)
	})
	g.Go(func() error {
		defer func() {
			for range out {
			}
		}()

		for m := range out {
			err := stream.Send(&pb.RunResponse{Proposal: m})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return g.Wait()
}

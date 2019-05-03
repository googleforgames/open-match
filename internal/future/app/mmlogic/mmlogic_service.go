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

package mmlogic

import (
	"open-match.dev/open-match/internal/future/pb"
)

// The MMLogic API provides utility functions for common MMF functionality such
// as retreiving Tickets from state storage.
type mmlogicService struct {
}

// RetrievePool gets the list of Tickets that match every Filter in the
// specified Pool.
// TODO: Consider renaming to "GetPool" to be consistent with HTTP REST CRUD
// conventions. Right now there's a GET and a POST for this verb.
func (s *mmlogicService) RetrievePool(req *pb.RetrievePoolRequest, stream pb.MmLogic_RetrievePoolServer) error {
	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			return nil

		default:
			err := stream.Send(&pb.RetrievePoolResponse{})
			if err != nil {
				return err
			}
		}
	}
}

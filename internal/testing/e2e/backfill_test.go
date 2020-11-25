// Copyright 2020 Google LLC
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

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"open-match.dev/open-match/pkg/pb"
)

// TestAcknowledgeBackfill Update Backfill test
func TestAcknowledgeBackfill(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	bf := &pb.Backfill{SearchFields: &pb.SearchFields{
		StringArgs: map[string]string{
			"search": "me",
		},
	},
	}
	createdBf, err := om.Frontend().CreateBackfill(ctx, &pb.CreateBackfillRequest{Backfill: bf})
	require.Nil(t, err)

	acknowledgedBf, err := om.Frontend().AcknowledgeBackfill(ctx, &pb.AcknowledgeBackfillRequest{BackfillId: createdBf.Id, Assignment: &pb.Assignment{Connection: "127.0.0.1:54000"}})
	require.Nil(t, err)
	require.Equal(t, acknowledgedBf.Id, createdBf.Id)
}

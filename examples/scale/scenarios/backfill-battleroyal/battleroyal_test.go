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

package battleroyal

import (
	"reflect"
	"testing"

	"open-match.dev/open-match/pkg/pb"
)

func TestBackfillScenario_MatchFunction(t *testing.T) {
	type fields struct {
		regions int
	}
	type args struct {
		p             *pb.MatchProfile
		poolTickets   map[string][]*pb.Ticket
		poolBackfills map[string][]*pb.Backfill
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*pb.Match
		wantErr bool
	}{
		{
			name: "Happy test",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			b := &BackfillScenario{
				regions: tt.fields.regions,
			}
			got, err := b.MatchFunction(tt.args.p, tt.args.poolTickets, tt.args.poolBackfills)
			if (err != nil) != tt.wantErr {
				t.Errorf("BackfillScenario.MatchFunction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BackfillScenario.MatchFunction() = %v, want %v", got, tt.want)
			}
		})
	}
}

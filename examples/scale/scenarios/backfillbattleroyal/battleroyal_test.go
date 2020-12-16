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

package backfillbattleroyal

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"open-match.dev/open-match/pkg/pb"
)

func TestBackfillScenario_MatchFunction(t *testing.T) {
	type fields struct {
		regions int
	}

	tickets := []*pb.Ticket{}
	backfills := []*pb.Backfill{}

	for i := 0; i < 100; i++ {
		tickets = append(tickets, &pb.Ticket{
			SearchFields: &pb.SearchFields{
				StringArgs: map[string]string{
					regionArg: "all",
				},
			}})
	}

	type args struct {
		p             *pb.MatchProfile
		poolTickets   map[string][]*pb.Ticket
		poolBackfills map[string][]*pb.Backfill
	}
	bf := &pb.Backfill{SearchFields: tickets[0].SearchFields}
	backfills = append(backfills, bf)
	tests := []struct {
		name         string
		args         args
		want         int
		wantErr      bool
		wantBackfill *pb.Backfill
	}{
		{
			name: "First round of MMF",
			args: args{
				p:             &pb.MatchProfile{},
				poolTickets:   map[string][]*pb.Ticket{"all": tickets},
				poolBackfills: map[string][]*pb.Backfill{},
			},
			want:         1,
			wantBackfill: bf,
		},
		{
			name: "Second round of MMF",
			args: args{
				p:             &pb.MatchProfile{},
				poolTickets:   map[string][]*pb.Ticket{"all": tickets},
				poolBackfills: map[string][]*pb.Backfill{"all": backfills},
			},
			want:         1,
			wantBackfill: bf,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			b := &BackfillScenario{
				regions: 0,
			}
			got, err := b.MatchFunction(tt.args.p, tt.args.poolTickets, tt.args.poolBackfills)
			if (err != nil) != tt.wantErr {
				t.Errorf("BackfillScenario.MatchFunction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.Len(t, got, tt.want)
			if tt.want > 0 && !reflect.DeepEqual(got[0].Backfill, tt.wantBackfill) {
				t.Errorf("BackfillScenario.MatchFunction() = %v, want %v", got, tt.want)
			}
			/*
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("BackfillScenario.MatchFunction() = %v, want %v", got, tt.want)
				} */
		})
	}
}

func newSearchFields(pool *pb.Pool) *pb.SearchFields {
	searchFields := pb.SearchFields{}
	rangeFilters := pool.GetDoubleRangeFilters()

	if rangeFilters != nil {
		doubleArgs := make(map[string]float64)
		for _, f := range rangeFilters {
			doubleArgs[f.DoubleArg] = (f.Max - f.Min) / 2
		}

		if len(doubleArgs) > 0 {
			searchFields.DoubleArgs = doubleArgs
		}
	}

	stringFilters := pool.GetStringEqualsFilters()

	if stringFilters != nil {
		stringArgs := make(map[string]string)
		for _, f := range stringFilters {
			stringArgs[f.StringArg] = f.Value
		}

		if len(stringArgs) > 0 {
			searchFields.StringArgs = stringArgs
		}
	}

	tagFilters := pool.GetTagPresentFilters()

	if tagFilters != nil {
		tags := make([]string, len(tagFilters))
		for _, f := range tagFilters {
			tags = append(tags, f.Tag)
		}

		if len(tags) > 0 {
			searchFields.Tags = tags
		}
	}

	return &searchFields
}

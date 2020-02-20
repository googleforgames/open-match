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

// Package filter defines which tickets pass which filters.  Other implementations which help
// filter tickets (eg, a range index lookup) must conform to the same set of tickets being within
// the filter as here.
package filter

import (
	"open-match.dev/open-match/pkg/pb"
)

var emptySearchFields = &pb.SearchFields{}

// InPool returns whether the ticket meets all the criteria of the pool.
func InPool(ticket *pb.Ticket, pool *pb.Pool) bool {
	s := ticket.GetSearchFields()
	if s == nil {
		s = emptySearchFields
	}
	for _, f := range pool.GetDoubleRangeFilters() {
		v, ok := s.DoubleArgs[f.DoubleArg]
		if !ok {
			return false
		}
		// Not simplified so that NaN cases are handled correctly.
		if !(v >= f.Min && v <= f.Max) {
			return false
		}
	}

	for _, f := range pool.GetStringEqualsFilters() {
		v, ok := s.StringArgs[f.StringArg]
		if !ok {
			return false
		}
		if f.Value != v {
			return false
		}
	}

outer:
	for _, f := range pool.GetTagPresentFilters() {
		for _, v := range s.Tags {
			if v == f.Tag {
				continue outer
			}
		}
		return false
	}

	return true
}

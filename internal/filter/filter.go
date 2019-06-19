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
	structpb "github.com/golang/protobuf/ptypes/struct"
	"open-match.dev/open-match/pkg/pb"
)

// Filter returns all of the passed tickets which are included by all the passed filters.
func Filter(tickets []*pb.Ticket, filters []*pb.Filter) []*pb.Ticket {
	var result []*pb.Ticket

	for _, t := range tickets {
		if InFilters(t, filters) {
			result = append(result, t)
		}
	}

	return result
}

// InFilters returns if the given ticket is included in all of the passed filters.
func InFilters(ticket *pb.Ticket, filters []*pb.Filter) bool {
	for _, f := range filters {
		if !InFilter(ticket, f) {
			return false
		}
	}
	return true
}

// InFilter returns if the given ticket is included in the given filter.
func InFilter(ticket *pb.Ticket, filter *pb.Filter) bool {
	if ticket.GetProperties().GetFields() == nil {
		return false
	}

	v, ok := ticket.Properties.Fields[filter.Attribute]
	if !ok {
		return false
	}

	switch v.Kind.(type) {
	case *structpb.Value_NumberValue:
		nv := v.GetNumberValue()
		return filter.Min <= nv && nv <= filter.Max
	default:
		return false
	}
}

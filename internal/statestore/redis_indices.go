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

package statestore

import (
	"math"
	"strings"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/pkg/pb"
)

// This file translates between the Open Match API's concept of fields and
// filters to a concept compatible with redis.  All indicies in redis are,
// for simplicity, sorted sets.  The following translations are used:
// Float values to range indicies use the float value directly, with filters
// doing direct lookups on those ranges.
// Boolean values with bool equals indicies turn true and false into 1 and 0.
// Filters on bool equality use ranges limited to only 1s or only 0s.
// Strings values are indexed by a unique attribute/value pair with value 0.
// Filters are strings look up that attribute/value pair.

func extractIndexedFields(cfg config.View, t *pb.Ticket) map[string]float64 {
	result := make(map[string]float64)

	for arg, value := range t.GetSearchFields().GetStringArgs() {
		result[stringIndexName(arg, value)] = 0
	}

	for _, tag := range t.GetSearchFields().GetTags() {
		result[tagIndexName(tag)] = 0
	}

	var indices []string
	if cfg.IsSet("ticketIndices") {
		indices = cfg.GetStringSlice("ticketIndices")
	}

	for _, attribute := range indices {
		v, ok := t.GetProperties().GetFields()[attribute]

		if !ok {
			redisLogger.WithFields(logrus.Fields{
				"attribute": attribute}).Trace("Couldn't find index in Ticket Properties")
			continue
		}

		switch v.Kind.(type) {
		case *structpb.Value_NumberValue:
			result[rangeIndexName(attribute)] = v.GetNumberValue()
		default:
			redisLogger.WithFields(logrus.Fields{
				"attribute": attribute,
			}).Warning("Attribute indexed but is not an expected type.")
		}
	}

	result[allTickets] = 0

	return result
}

type indexFilter struct {
	name     string
	min, max float64
}

func extractIndexFilters(p *pb.Pool) []indexFilter {
	filters := make([]indexFilter, 0)

	for _, f := range p.FloatRangeFilters {
		filters = append(filters, indexFilter{
			name: rangeIndexName(f.Attribute),
			min:  f.Min,
			max:  f.Max,
		})
	}

	for _, f := range p.TagPresentFilters {
		filters = append(filters, indexFilter{
			name: tagIndexName(f.Tag),
			min:  0,
			max:  0,
		})
	}

	for _, f := range p.StringEqualsFilters {
		filters = append(filters, indexFilter{
			name: stringIndexName(f.Attribute, f.Value),
			min:  0,
			max:  0,
		})
	}

	if len(filters) == 0 {
		filters = []indexFilter{{
			name: allTickets,
			min:  math.Inf(-1),
			max:  math.Inf(1),
		}}
	}

	return filters
}

// The following are constants and functions for determining the names of
// indices.  Different index types have different prefixes to avoid any
// name collision.
const allTickets = "allTickets"

func rangeIndexName(attribute string) string {
	// ri stands for range index
	return "ri$" + indexEscape(attribute)
}

func tagIndexName(attribute string) string {
	// ti stands for tag index
	return "ti$" + indexEscape(attribute)
}

func stringIndexName(attribute string, value string) string {
	// si stands for string index
	return "si$" + indexEscape(attribute) + "$v" + indexEscape(value)
}

func indexCacheName(id string) string {
	// ic stands for index cache
	return "ic$" + indexEscape(id)
}

func indexEscape(s string) string {
	return strings.ReplaceAll(s, "$", "$$")
}

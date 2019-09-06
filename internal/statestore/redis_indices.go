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
	"net/url"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/pkg/pb"
)

type indexedFields struct {
	values map[string]float64
}

func extractIndexedFields(cfg config.View, t *pb.Ticket) indexedFields {
	result := indexedFields{
		values: make(map[string]float64),
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
			result.values[rangeIndexName(attribute)] = v.GetNumberValue()
		default:
			redisLogger.WithFields(logrus.Fields{
				"attribute": attribute,
			}).Warning("Attribute indexed but is not a number.")
		}
	}

	result[allTickets] = float64(time.Now().UnixNano())

	return result
}

type indexFilter struct {
	name     string
	min, max float64
}

func extractIndexFilters(p *pb.Pool) []indexFilter {
	filters := make([]indexFilter, 0)

	for _, f := range p.Filters {
		filters = append(filters, indexFilter{
			name: rangeIndexName(f.Attribute),
			min:  f.Min,
			max:  f.Max,
		})
	}

	if len(filters) == 0 {
		filters = []indexFilter{{
			name: allTickets,
			min:  math.Inf(-1),
			min:  math.Inf(1),
		}}
	}

	return filters
}

func extractDeindexFilters(cfg config.View) []string {
	var indices []string
	if cfg.IsSet("ticketIndices") {
		indices = cfg.GetStringSlice("ticketIndices")
	}

	result := []string{allTickets}

	for _, index := range indices {
		result = append(result, rangeIndexName(index))
	}

	return result
}

// The following are constants and functions for determining the names of
// indexes.  Different index types have different prefixes to avoid any
// name collision.
const allTickets = "allTickets"

func rangeIndexName(attribute string) string {
	// ri stands for range index
	return "ri$" + url.QueryEscape(attribute)
}

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
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"open-match.dev/open-match/pkg/pb"
)

var emptySearchFields = &pb.SearchFields{}

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "filter",
	})
)

// PoolFilter contains all the filtering criteria from a Pool that the Ticket
// needs to meet to belong to that Pool.
type PoolFilter struct {
	DoubleRangeFilters  []*pb.DoubleRangeFilter
	StringEqualsFilters []*pb.StringEqualsFilter
	TagPresentFilters   []*pb.TagPresentFilter
	CreatedBefore       time.Time
	CreatedAfter        time.Time
}

// NewPoolFilter validates a Pool's filtering criteria and returns a PoolFilter.
func NewPoolFilter(pool *pb.Pool) (*PoolFilter, error) {
	var ca, cb time.Time
	var err error

	if pool.GetCreatedBefore() != nil {
		if err = pool.GetCreatedBefore().CheckValid(); err != nil {
			return nil, status.Error(codes.InvalidArgument, ".invalid created_before value")
		}
		cb = pool.GetCreatedBefore().AsTime()
	}

	if pool.GetCreatedAfter() != nil {
		if err = pool.GetCreatedAfter().CheckValid(); err != nil {
			return nil, status.Error(codes.InvalidArgument, ".invalid created_after value")
		}
		ca = pool.GetCreatedAfter().AsTime()
	}

	return &PoolFilter{
		DoubleRangeFilters:  pool.GetDoubleRangeFilters(),
		StringEqualsFilters: pool.GetStringEqualsFilters(),
		TagPresentFilters:   pool.GetTagPresentFilters(),
		CreatedBefore:       cb,
		CreatedAfter:        ca,
	}, nil
}

type filteredEntity interface {
	GetId() string
	GetSearchFields() *pb.SearchFields
	GetCreateTime() *timestamppb.Timestamp
}

// In returns true if the Ticket meets all the criteria for this PoolFilter.
func (pf *PoolFilter) In(entity filteredEntity) bool {
	s := entity.GetSearchFields()

	if s == nil {
		s = emptySearchFields
	}

	if !pf.CreatedAfter.IsZero() || !pf.CreatedBefore.IsZero() {
		// CreateTime is only populated by Open Match and hence expected to be valid.
		if err := entity.GetCreateTime().CheckValid(); err == nil {
			ct := entity.GetCreateTime().AsTime()
			if !pf.CreatedAfter.IsZero() {
				if !ct.After(pf.CreatedAfter) {
					return false
				}
			}

			if !pf.CreatedBefore.IsZero() {
				if !ct.Before(pf.CreatedBefore) {
					return false
				}
			}
		} else {
			logger.WithFields(logrus.Fields{
				"error": err.Error(),
				"id":    entity.GetId(),
			}).Error("failed to get time from Timestamp proto")
		}
	}

	for _, f := range pf.DoubleRangeFilters {
		v, ok := s.DoubleArgs[f.DoubleArg]
		if !ok {
			return false
		}

		switch f.Exclude {
		case pb.DoubleRangeFilter_NONE:
			// Not simplified so that NaN cases are handled correctly.
			if !(v >= f.Min && v <= f.Max) {
				return false
			}
		case pb.DoubleRangeFilter_MIN:
			if !(v > f.Min && v <= f.Max) {
				return false
			}
		case pb.DoubleRangeFilter_MAX:
			if !(v >= f.Min && v < f.Max) {
				return false
			}
		case pb.DoubleRangeFilter_BOTH:
			if !(v > f.Min && v < f.Max) {
				return false
			}
		}

	}

	for _, f := range pf.StringEqualsFilters {
		v, ok := s.StringArgs[f.StringArg]
		if !ok {
			return false
		}
		if f.Value != v {
			return false
		}
	}

outer:
	for _, f := range pf.TagPresentFilters {
		for _, v := range s.Tags {
			if v == f.Tag {
				continue outer
			}
		}
		return false
	}

	return true
}

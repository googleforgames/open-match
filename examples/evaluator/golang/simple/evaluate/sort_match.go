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

package evaluate

import (
	"sort"

	"open-match.dev/open-match/pkg/pb"
)

// by is the type of a "less" function that defines the ordering of its Planet arguments.
type by func(p1, p2 *pb.Match) bool

// matchSorter joins a By function and a slice of Matches to be sorted.
type matchSorter struct {
	matches []*pb.Match
	by      func(a, b *pb.Match) bool // Closure used in the Less method.
}

// Sort is a method on the function type, By, that sorts the argument slice according to the function.
func (by by) Sort(matches []*pb.Match) {
	sort.Sort(&matchSorter{
		matches: matches,
		by:      by, // The Sort method's receiver is the function (closure) that defines the sort order.
	})
}

// Len is part of sort.Interface.
func (s *matchSorter) Len() int {
	return len(s.matches)
}

// Swap is part of sort.Interface.
func (s *matchSorter) Swap(i, j int) {
	s.matches[i], s.matches[j] = s.matches[j], s.matches[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *matchSorter) Less(i, j int) bool {
	return s.by(s.matches[i], s.matches[j])
}

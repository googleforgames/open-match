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

package e2e

var (
	zygote OM

	// Indices covers all ticketIndices that will be used in the e2e test
	// Please update the ticketIndices in the helm chart for in-cluster end-to-end test if you need to add new indices.
	Indices = []string{FloatRangeIndex1, FloatRangeIndex2, FloatRangeIndex3, BoolIndex1, StringIndex1}
)

const (
	// Map1BeginnerPool is a pool name.
	Map1BeginnerPool = "map1beginner"
	// Map1AdvancedPool is pool name.
	Map1AdvancedPool = "map1advanced"
	// Map2BeginnerPool is pool name.
	Map2BeginnerPool = "map2beginner"
	// Map2AdvancedPool is pool name.
	Map2AdvancedPool = "map2advanced"
	// FloatRangeIndex1 is an index used to test FloatRangeFilter.
	FloatRangeIndex1 = "attribute.mmr"
	// FloatRangeIndex2 is an index used to test FloatRangeFilter.
	FloatRangeIndex2 = "attribute.level"
	// FloatRangeIndex3 is an index used to test FloatRangeFilter.
	FloatRangeIndex3 = "attribute.defense"
	// BoolIndex1 is an index used to test BoolEqualsFilter
	BoolIndex1 = "mode.demo"
	// StringIndex1 is an index used to test StringEqualsFilter
	StringIndex1 = "char.cleric"
)

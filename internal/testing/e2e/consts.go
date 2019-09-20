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
	Indices = []string{AttributeMMR, AttributeLevel, AttributeDefense, ModeDemo, RoleCleric}
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
	// AttributeMMR is an index used to test FloatRangeFilter.
	AttributeMMR = "attribute.mmr"
	// AttributeLevel is an index used to test FloatRangeFilter.
	AttributeLevel = "attribute.level"
	// AttributeDefense is an index used to test FloatRangeFilter.
	AttributeDefense = "attribute.defense"
	// ModeDemo is an index used to test BoolEqualsFilter
	ModeDemo = "mode.demo"
	// RoleCleric is an index used to test StringEqualsFilter
	RoleCleric = "char.cleric"
)

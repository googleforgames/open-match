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
	// Please update #L394 in the Makefile if you need to add new indices to the e2e tests.
	Indices = []string{SkillAttribute, Map1Attribute, Map2Attribute, LevelIndex, DefenseIndex}
)

const (
	// Map1BeginnerPool is an index.
	Map1BeginnerPool = "map1beginner"
	// Map1AdvancedPool is an index.
	Map1AdvancedPool = "map1advanced"
	// Map2BeginnerPool is an index.
	Map2BeginnerPool = "map2beginner"
	// Map2AdvancedPool is an index.
	Map2AdvancedPool = "map2advanced"
	// SkillAttribute is an index.
	SkillAttribute = "skill"
	// Map1Attribute is an index.
	Map1Attribute = "map1"
	// Map2Attribute is an index.
	Map2Attribute = "map2"
	// LevelIndex is an index
	LevelIndex = "level"
	// DefenseIndex is an index
	DefenseIndex = "defense"
)

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

package internal

import (
	"fmt"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsOrphaned(t *testing.T) {
	testCases := []struct {
		age      time.Duration
		label    string
		cluster  *containerpb.Cluster
		expected bool
	}{
		{
			time.Second,
			"ci",
			&containerpb.Cluster{
				Name:           "Orphaned",
				CreateTime:     time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				ResourceLabels: map[string]string{"ci": "ok"},
			},
			true,
		},
		{
			time.Second,
			"ci",
			&containerpb.Cluster{
				Name:           "Not Orphaned, Created in the Future",
				CreateTime:     time.Now().Add(1 * time.Hour).Format(time.RFC3339),
				ResourceLabels: map[string]string{"ci": "ok"},
			},
			false,
		},
		{
			time.Second,
			"ci",
			&containerpb.Cluster{
				Name:           "Not Orphaned, Created in Past but no Label",
				CreateTime:     time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				ResourceLabels: map[string]string{},
			},
			false,
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s deleted= %t", tc.cluster.Name, tc.expected), func(t *testing.T) {
			actual := isOrphaned(tc.cluster, &Params{
				Age:   tc.age,
				Label: tc.label,
			})
			if actual != tc.expected {
				if tc.expected {
					t.Errorf("Cluster %s was NOT marked as orphaned but should have.", tc.cluster.Name)
				} else {
					t.Errorf("Cluster %s was marked as orphaned but should not have.", tc.cluster.Name)
				}
			}
		})
	}
}

func TestSelfLinkToQualifiedName(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("projects/jeremyedwards-gaming-dev/zones/us-central1-a/clusters/orphaned", selfLinkToQualifiedName("https://container.googleapis.com/v1/projects/jeremyedwards-gaming-dev/zones/us-central1-a/clusters/orphaned"))
}

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

// Package internal holds the internal workings for deleting orphaned GKE clusters from Continuous Integration.
package internal

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	container "cloud.google.com/go/container/apiv1"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

// Params for deleting a cluster.
type Params struct {
	Age       time.Duration
	Label     string
	ProjectID string
	Location  string
}

// ReapCluster scans for orphaned clusters and deletes them.
func ReapCluster(params *Params) {
	log.Printf("Scanning for orphaned clusters in projects/%s/locations/%s that are older than %s with label %s.", params.ProjectID, params.Location, params.Age.String(), params.Label)
	ctx := context.Background()
	c, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	clusters, err := listOrphanedClusters(ctx, c, params)
	if err != nil {
		log.Fatal(err)
	}

	err = deleteClusters(ctx, c, clusters)
	if err != nil {
		log.Fatal(err)
	}

	if len(clusters) > 0 {
		log.Printf("Deleted %d GKE clusters", len(clusters))
	} else {
		log.Printf("There were no orphaned clusters. :)")
	}
}

func listOrphanedClusters(ctx context.Context, c *container.ClusterManagerClient, params *Params) ([]*containerpb.Cluster, error) {
	resp, err := c.ListClusters(ctx, &containerpb.ListClustersRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", params.ProjectID, params.Location),
	})
	if err != nil {
		return []*containerpb.Cluster{}, err
	}
	orphanedClusters := []*containerpb.Cluster{}
	for _, cluster := range resp.Clusters {
		if isOrphaned(cluster, params) {
			log.Printf("Cluster '/%s' was orphaned.\nDetails: %+v", cluster.Name, cluster)
			orphanedClusters = append(orphanedClusters, cluster)
		}
	}
	return orphanedClusters, nil
}

func deleteClusters(ctx context.Context, c *container.ClusterManagerClient, clusters []*containerpb.Cluster) error {
	for _, cluster := range clusters {
		clusterName := selfLinkToQualifiedName(cluster.SelfLink)
		log.Printf("Deleting Orphaned GKE Cluster %s", clusterName)
		resp, err := c.DeleteCluster(ctx, &containerpb.DeleteClusterRequest{
			Name: clusterName,
		})
		if err != nil {
			return err
		}
		log.Printf("Delete GKE Cluster: %s => %v", clusterName, resp)
	}
	return nil
}

func isOrphaned(cluster *containerpb.Cluster, params *Params) bool {
	deadline := time.Now().Add(-1 * params.Age)
	creationTime, err := time.Parse(time.RFC3339, cluster.CreateTime)
	if err != nil {
		log.Printf("cannot parse time '%s' because '%v', ignoring for orphan detection.", cluster.CreateTime, err)
		return false
	}
	if creationTime.After(deadline) {
		return false
	}
	if _, ok := cluster.ResourceLabels[params.Label]; !ok {
		return false
	}
	return true
}

func selfLinkToQualifiedName(selfLink string) string {
	parts := strings.Split(selfLink, "v1/")
	if len(parts) != 2 {
		log.Fatalf("SelfLink: %s is not a valid selflink it should contain 'v1/'", selfLink)
	}
	return parts[1]
}

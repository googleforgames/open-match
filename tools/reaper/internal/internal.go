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
	container "cloud.google.com/go/container/apiv1"
	"context"
	"fmt"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// Params for deleting a cluster.
type Params struct {
	Age       time.Duration
	Label     string
	ProjectID string
	Location  string
}

func portFromListener(conn net.Listener) int {
	tcpConn, ok := conn.Addr().(*net.TCPAddr)
	if !ok || tcpConn == nil {
		log.Fatalf("net.Listen(\"tcp\", %+v) did not return a *net.TCPAddr", conn.Addr())
	}
	return tcpConn.Port
}

// Serve brings up an HTTP server and waits for a request.
// It serves 3 endpoints:
// /livenessz - for liveness/readiness check
// /reap      - to delete the clusters
// /close     - to shutdown the application.
// The intention of this serving mode is so that this tool can run from Google Cloud Run.
func Serve(conn net.Listener, params *Params) error {
	port := portFromListener(conn)
	addr := fmt.Sprintf(":%d", port)

	server := &http.Server{
		Addr: addr,
	}

	mux := &http.ServeMux{}
	mux.HandleFunc("/livenessz", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprint(w, "OK")
	})
	mux.HandleFunc("/close", func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := server.Close(); err != nil {
				log.Printf("Unclean shutdown, %s", err)
			}
		}()
		fmt.Fprint(w, "OK")
	})

	mux.HandleFunc("/reap", func(w http.ResponseWriter, req *http.Request) {
		resp, err := ReapCluster(params)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "ERROR: %s", err.Error())
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK: %s", resp)
		}
	})

	server.Handler = mux
	log.Printf("Serving on %s", addr)
	return server.Serve(conn)
}

// ReapCluster scans for orphaned clusters and deletes them.
func ReapCluster(params *Params) (string, error) {
	log.Printf("Scanning for orphaned clusters in projects/%s/locations/%s that are older than %s with label %s.", params.ProjectID, params.Location, params.Age.String(), params.Label)
	ctx := context.Background()
	c, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return "", err
	}
	clusters, err := listOrphanedClusters(ctx, c, params)
	if err != nil {
		return "", err
	}

	err = deleteClusters(ctx, c, clusters)
	if err != nil {
		return "", err
	}

	if len(clusters) > 0 {
		return fmt.Sprintf("Deleted %d GKE clusters", len(clusters)), nil
	}
	return fmt.Sprintf("There were no orphaned clusters. :)"), nil
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

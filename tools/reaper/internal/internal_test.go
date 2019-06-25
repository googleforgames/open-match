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
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServeAndClose(t *testing.T) {
	conn, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("cannot open TCP port, %s", err)
	}
	serve(conn)
	port := portFromListener(conn)
	killLater(port)

	if resp, err := http.Get(fmt.Sprintf("http://localhost:%d/livenessz", port)); err != nil {
		t.Errorf("/livenessz error %s", err)
	} else {
		if text, err := ioutil.ReadAll(resp.Body); err != nil {
			t.Errorf("/livenessz read body error %s", err)
		} else if string(text) != "OK" {
			t.Errorf("/livenessz expected 'OK' got %s", text)
		}
	}
}

func serve(conn net.Listener) func() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		err := Serve(conn, &Params{
			Age:       time.Minute * 30,
			Label:     "open-match-ci",
			ProjectID: "open-match-build",
			Location:  "us-west1-a",
		})
		if err != nil {
			fmt.Printf("http server did not shutdown cleanly, %s", err)
		}
	}()
	return wg.Wait
}

func killLater(port int) {
	go func() {
		time.Sleep(500 * time.Millisecond)
		_, err := http.Get(fmt.Sprintf("http://localhost:%d/close", port))
		if err != nil {
			fmt.Printf("Got error closing server: %s", err)
		}
	}()
}

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
	assert.Equal("projects/open-match-build/zones/us-central1-a/clusters/orphaned", selfLinkToQualifiedName("https://container.googleapis.com/v1/projects/open-match-build/zones/us-central1-a/clusters/orphaned"))
}

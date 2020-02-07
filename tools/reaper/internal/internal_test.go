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
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServeAndClose(t *testing.T) {
	conn, err := net.Listen("tcp", "localhost:0")
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
			Age: time.Minute * 30,
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
		age       time.Duration
		namespace corev1.Namespace
		expected  bool
	}{
		{
			time.Second,
			corev1.Namespace{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour).UTC()},
					Name:              "open-match-test-namespace",
				},
			},
			true,
		},
		{
			time.Second,
			corev1.Namespace{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: time.Now().Add(1 * time.Hour).UTC()},
					Name:              "open-match-future-namespace",
				},
			},
			false,
		},
		{
			time.Second,
			corev1.Namespace{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour).UTC()},
					Name:              "kube-system",
				},
			},
			false,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("%s deleted= %t", tc.namespace.ObjectMeta.Name, tc.expected), func(t *testing.T) {
			actual := isOrphaned(&tc.namespace, &Params{
				Age: tc.age,
			})
			if actual != tc.expected {
				if tc.expected {
					t.Errorf("Namespace %s was NOT marked as orphaned but should have.", tc.namespace.ObjectMeta.Name)
				} else {
					t.Errorf("Namespace %s was marked as orphaned but should not have.", tc.namespace.ObjectMeta.Name)
				}
			}
		})
	}
}

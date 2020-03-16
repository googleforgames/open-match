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
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	// Enabled gcloud auth plugin
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/pkg/errors"
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
		err := ReapNamespaces(params)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "ERROR: %s", err.Error())
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK")
		}
	})

	server.Handler = mux
	log.Printf("Serving on %s", addr)
	return server.Serve(conn)
}

// ReapNamespaces scans for orphaned namespaces and deletes them.
func ReapNamespaces(params *Params) error {
	log.Printf("Scanning for orphaned namespaces that are older than %s.", params.Age.String())

	u, err := user.Current()
	if err != nil {
		log.Fatalf("cannot get current user, %s", err)
	}

	kubeconfig := filepath.Join(u.HomeDir, ".kube", "config")
	kubeconfigFromEnv := os.Getenv("KUBECONFIG")

	if !fileExists(kubeconfig) && fileExists(kubeconfigFromEnv) {
		kubeconfig = kubeconfigFromEnv
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return errors.Wrapf(err, "building Kubernetes config from flags %s failed", kubeconfig)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrapf(err, "creating Kubernetes client from config failed\nkubeconfig= %s\nconfig= %+v", kubeconfig, config)
	}

	namespaceInterface := kubeClient.CoreV1().Namespaces()
	namespaces, err := namespaceInterface.List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "listing namespace from kubernetes cluster failed")
	}

	for i := 0; i < len(namespaces.Items); i++ {
		namespace := namespaces.Items[i]
		if !isOrphaned(&namespace, params) {
			log.Printf("skipping namespace %s created at %s", namespace.ObjectMeta.Name, namespace.ObjectMeta.CreationTimestamp)
			continue
		}

		err = namespaceInterface.Delete(namespace.ObjectMeta.Name, &metav1.DeleteOptions{})
		if err != nil {
			log.Printf("error: %s", err.Error())
			continue
		}

		log.Printf("deleting namespace %s created at %s", namespace.ObjectMeta.Name, namespace.ObjectMeta.CreationTimestamp)
	}

	return err
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func isOrphaned(namespace *corev1.Namespace, params *Params) bool {
	deadline := time.Now().Add(-1 * params.Age).UTC()
	return namespace.ObjectMeta.CreationTimestamp.Time.Before(deadline) && strings.HasPrefix(namespace.ObjectMeta.Name, "open-match")
}

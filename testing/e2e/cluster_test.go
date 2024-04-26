//go:build e2ecluster
// +build e2ecluster

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

import (
	"context"
	"fmt"
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"open-match.dev/open-match/internal/app/evaluator"
	"open-match.dev/open-match/internal/appmain/apptest"
	"open-match.dev/open-match/pkg/pb"
	mmfService "open-match.dev/open-match/testing/mmf"
)

func TestServiceHealth(t *testing.T) {
	kubeconfig, err := rest.InClusterConfig()
	if err != nil {
		t.Fatal(err)
	}

	kubeClient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		t.Fatalf("%s: creating Kubernetes client from config failed\nconfig= %+v", err, kubeconfig)
	}

	namespace := os.Getenv("NAMESPACE")

	podList, err := kubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, pod := range podList.Items {
		if app, ok := pod.ObjectMeta.Labels["app"]; ok && app == "open-match" && pod.Status.Phase != corev1.PodRunning {
			t.Errorf("pod %+v is not running.", pod)
		}
	}
}

func TestMain(m *testing.M) {
	clusterStarted = true
	mmf := func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
		return clusterMMF(ctx, profile, out)
	}

	eval := func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
		return clusterEval(ctx, in, out)
	}

	cleanup, err := apptest.RunInCluster(mmfService.BindServiceFor(mmf), evaluator.BindServiceFor(eval))
	if err != nil {
		fmt.Println("Error starting mmf and evaluator:", err)
		os.Exit(1)
	}

	exitCode := m.Run()

	err = cleanup()
	if err != nil {
		fmt.Println("Error stopping mmf and evaluator:", err)
		os.Exit(1)
	}

	os.Exit(exitCode)
}

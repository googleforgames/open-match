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

package framework

import (
	"flag"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

var (
	// One time init variable protected by Get().
	framework *Framework
)

// Framework holds the details for connecting to a Kubernetes cluster.
type Framework struct {
	KubeClient kubernetes.Interface
	Namespace  string
}

// Cleanup deletes all resources associated with the run.
func (f *Framework) Cleanup() error {
	return nil
}

// Get the framework object.
func Get() *Framework {
	return framework
}

func newFramework(kubeconfig string, namespace string) (*Framework, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "build config from flags failed")
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "creating new kube-client failed")
	}

	return &Framework{
		KubeClient: kubeClient,
		Namespace:  namespace,
	}, nil
}

// RunMain provides the setup and teardown for Open Match e2e tests.
func RunMain(m *testing.M) {
	var exitCode int
	u, err := user.Current()
	if err != nil {
		log.Fatalf("cannot get user context, %s", err)
	}
	kubeconfig := flag.String("kubeconfig", filepath.Join(u.HomeDir, ".kube", "config"), "Kubernetes configuration file")
	namespace := flag.String("namespace", "open-match", "Open Match Namespace")

	flag.Parse()
	f, err := newFramework(*kubeconfig, *namespace)
	if err != nil {
		log.Fatalf("failed to setup framework: %s", err)
	}
	defer func() {
		cErr := f.Cleanup()
		if cErr != nil {
			log.Printf("failed to cleanup resources: %s", cErr)
		}
		os.Exit(exitCode)
	}()
	framework = f
	exitCode = m.Run()
}

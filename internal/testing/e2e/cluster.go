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
	"flag"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/internal/util"
	pb "open-match.dev/open-match/pkg/pb"
)

type clusterOM struct {
	kubeClient kubernetes.Interface
	namespace  string
	t          *testing.T
	mc         *util.MultiClose
}

func (com *clusterOM) withT(t *testing.T) OM {
	return &clusterOM{
		kubeClient: com.kubeClient,
		namespace:  com.namespace,
		t:          t,
		mc:         util.NewMultiClose(),
	}
}

func (com *clusterOM) MustFrontendGRPC() pb.FrontendClient {
	conn, err := com.getGRPCClientFromServiceName("om-frontend")
	if err != nil {
		com.t.Fatalf("cannot create gRPC client, %s", err)
	}
	com.mc.AddCloseWithErrorFunc(conn.Close)
	return pb.NewFrontendClient(conn)
}

func (com *clusterOM) MustBackendGRPC() pb.BackendClient {
	conn, err := com.getGRPCClientFromServiceName("om-backend")
	if err != nil {
		com.t.Fatalf("cannot create gRPC client, %s", err)
	}
	com.mc.AddCloseWithErrorFunc(conn.Close)
	return pb.NewBackendClient(conn)
}

func (com *clusterOM) MustMmLogicGRPC() pb.MmLogicClient {
	conn, err := com.getGRPCClientFromServiceName("om-mmlogic")
	if err != nil {
		com.t.Fatalf("cannot create gRPC client, %s", err)
	}
	com.mc.AddCloseWithErrorFunc(conn.Close)
	return pb.NewMmLogicClient(conn)
}

func (com *clusterOM) MustMmfConfigGRPC() *pb.FunctionConfig {
	host, port := com.getGRPCAddressFromServiceName("om-e2ematchfunction")
	return &pb.FunctionConfig{
		Host: host,
		Port: port,
		Type: pb.FunctionConfig_GRPC,
	}
}

func (com *clusterOM) MustMmfConfigHTTP() *pb.FunctionConfig {
	host, port := com.getHTTPAddressFromServiceName("om-e2ematchfunction")
	return &pb.FunctionConfig{
		Host: host,
		Port: port,
		Type: pb.FunctionConfig_REST,
	}
}

func (com *clusterOM) getAddressFromServiceName(serviceName, portName string) (string, int32) {
	svc, err := com.kubeClient.CoreV1().Services(com.namespace).Get(serviceName, metav1.GetOptions{})
	if err != nil {
		com.t.Fatalf("cannot get service definition for %s", serviceName)
	}
	if len(svc.Status.LoadBalancer.Ingress) != 1 {
		com.t.Fatalf("LoadBalancer for %s does not have exactly 1 config, %v", serviceName, svc.Status.LoadBalancer.Ingress)
	}

	var port int32
	for _, servicePort := range svc.Spec.Ports {
		if servicePort.Name == portName {
			port = servicePort.Port
		}
	}
	return svc.Status.LoadBalancer.Ingress[0].IP, port
}

func (com *clusterOM) getGRPCAddressFromServiceName(serviceName string) (string, int32) {
	return com.getAddressFromServiceName(serviceName, "grpc")
}

func (com *clusterOM) getHTTPAddressFromServiceName(serviceName string) (string, int32) {
	return com.getAddressFromServiceName(serviceName, "http")
}

func (com *clusterOM) getGRPCClientFromServiceName(serviceName string) (*grpc.ClientConn, error) {
	ipAddress, port := com.getGRPCAddressFromServiceName(serviceName)
	conn, err := rpc.GRPCClientFromParams(&rpc.ClientParams{
		Hostname:         ipAddress,
		Port:             int(port),
		EnableRPCLogging: true,
		EnableMetrics:    false,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "cannot connect to gRPC %s:%d", ipAddress, port)
	}
	return conn, nil
}

func (com *clusterOM) HealthCheck() error {
	podList, err := com.kubeClient.CoreV1().Pods(com.namespace).List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "cannot get pods list")
	}
	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			return errors.Errorf("pod %+v is not running.", pod)
		}
	}
	return nil
}

func (com *clusterOM) Context() context.Context {
	return context.Background()
}

func (com *clusterOM) cleanup() {
	com.mc.Close()
}

func (com *clusterOM) cleanupMain() error {
	return nil
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func newClusterOM(kubeconfig string, kubeconfigFromEnv string, namespace string) (*clusterOM, error) {
	if !fileExists(kubeconfig) && fileExists(kubeconfigFromEnv) {
		kubeconfig = kubeconfigFromEnv
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, errors.Wrapf(err, "building Kubernetes config from flags %s failed", kubeconfig)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrapf(err, "creating Kubernetes client from config failed\nkubeconfig= %s\nconfig= %+v", kubeconfig, config)
	}

	return &clusterOM{
		kubeClient: kubeClient,
		namespace:  namespace,
	}, nil
}

func createZygote(m *testing.M) (OM, error) {
	u, err := user.Current()
	if err != nil {
		log.Fatalf("cannot get current user, %s", err)
	}
	kubeconfig := flag.String("kubeconfig", filepath.Join(u.HomeDir, ".kube", "config"), "Kubernetes configuration file")
	namespace := flag.String("namespace", "open-match", "Open Match Namespace")
	kubeconfigFromEnv := os.Getenv("KUBECONFIG")

	flag.Parse()
	return newClusterOM(*kubeconfig, kubeconfigFromEnv, *namespace)
}

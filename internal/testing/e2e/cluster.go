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
	"log"
	"os"
	"testing"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"open-match.dev/open-match/internal/logging"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/internal/util"

	pb "open-match.dev/open-match/pkg/pb"
)

func init() {
	// Reset the gRPC resolver to passthrough for end-to-end out-of-cluster testings.
	// DNS resolver is unsupported for end-to-end local testings.
	resolver.SetDefaultScheme("dns")
}

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

func (com *clusterOM) MustFrontendGRPC() pb.FrontendServiceClient {
	conn, err := com.getGRPCClientFromServiceName("om-frontend")
	if err != nil {
		com.t.Fatalf("cannot create gRPC client, %s", err)
	}
	com.mc.AddCloseWithErrorFunc(conn.Close)
	return pb.NewFrontendServiceClient(conn)
}

func (com *clusterOM) MustBackendGRPC() pb.BackendServiceClient {
	conn, err := com.getGRPCClientFromServiceName("om-backend")
	if err != nil {
		com.t.Fatalf("cannot create gRPC client, %s", err)
	}
	com.mc.AddCloseWithErrorFunc(conn.Close)
	return pb.NewBackendServiceClient(conn)
}

func (com *clusterOM) MustQueryServiceGRPC() pb.QueryServiceClient {
	conn, err := com.getGRPCClientFromServiceName("om-query")
	if err != nil {
		com.t.Fatalf("cannot create gRPC client, %s", err)
	}
	com.mc.AddCloseWithErrorFunc(conn.Close)
	return pb.NewQueryServiceClient(conn)
}

func (com *clusterOM) MustMmfConfigGRPC() *pb.FunctionConfig {
	host, port := com.getGRPCAddressFromServiceName("om-function")
	return &pb.FunctionConfig{
		Host: host,
		Port: port,
		Type: pb.FunctionConfig_GRPC,
	}
}

func (com *clusterOM) MustMmfConfigHTTP() *pb.FunctionConfig {
	host, port := com.getHTTPAddressFromServiceName("om-function")
	return &pb.FunctionConfig{
		Host: host,
		Port: port,
		Type: pb.FunctionConfig_REST,
	}
}

func (com *clusterOM) getAddressFromServiceName(serviceName, portName string) (string, int32) {
	endpoints, err := com.kubeClient.CoreV1().Endpoints(com.namespace).Get(serviceName, metav1.GetOptions{})
	if err != nil {
		com.t.Fatalf("cannot get service definition for %s, %s", serviceName, err.Error())
	}
	if len(endpoints.Subsets) == 0 || len(endpoints.Subsets[0].Addresses) == 0 {
		com.t.Fatalf("service %s does not have an available endpoint", serviceName)
	}

	var port int32
	for _, endpointsPort := range endpoints.Subsets[0].Ports {
		if endpointsPort.Name == portName {
			port = endpointsPort.Port
		}
	}
	return endpoints.Subsets[0].Addresses[0].IP, port
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
		Address:                 fmt.Sprintf("%s:%d", ipAddress, int(port)),
		EnableRPCLogging:        *testOnlyEnableRPCLoggingFlag,
		EnableRPCPayloadLogging: logging.IsDebugLevel(*testOnlyLoggingLevel),
		EnableMetrics:           *testOnlyEnableMetrics,
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
		if app, ok := pod.ObjectMeta.Labels["app"]; ok && app == "open-match" && pod.Status.Phase != corev1.PodRunning {
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

func createZygote(m *testing.M) (OM, error) {
	// creates the in-cluster config
	kubeconfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, errors.Wrapf(err, "creating Kubernetes client from config failed\nconfig= %+v", kubeconfig)
	}

	return &clusterOM{
		kubeClient: kubeClient,
		namespace:  os.Getenv("NAMESPACE"),
	}, nil
}

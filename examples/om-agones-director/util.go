package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getKubeConfig() *rest.Config {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	return config
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func getGRPCConnFromSvcName(cfg *rest.Config, svcName string) *grpc.ClientConn {
	conn, err := grpc.Dial(getAddrFromSvcName(cfg, svcName, "grpc"), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	return conn
}

func getAddrFromSvcName(cfg *rest.Config, svcName, connMode string) string {
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	svc, err := kubeClient.CoreV1().Services("open-match").Get(svcName, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	if len(svc.Status.LoadBalancer.Ingress) == 0 {
		panic(errors.New(fmt.Sprintf("service: %s does not have ingress exposed.\n", svcName)))
	}

	port := int32(-1)
	for _, servicePort := range svc.Spec.Ports {
		if servicePort.Name == connMode {
			port = servicePort.Port
		}
	}

	if port == -1 {
		panic(errors.New(fmt.Sprintf("connection port mode: %s not found", connMode)))
	}

	return fmt.Sprintf("%s:%d", svc.Status.LoadBalancer.Ingress[0].IP, port)
}

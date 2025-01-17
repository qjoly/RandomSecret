package kube

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

// init client
type (
	KubeClient struct {
		Clientset *kubernetes.Clientset
		Cfg       *rest.Config
	}
)

func NewClient() *KubeClient {
	client := &KubeClient{
		Cfg: getConfig(),
	}

	clientset, err := kubernetes.NewForConfig(client.Cfg)
	if err != nil {
		klog.Info(fmt.Sprintf("Error creating clientset: %v", err))
	}

	client.Clientset = clientset

	return client
}

func getConfig() *rest.Config {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		// If not running in a cluster, use a traditional kubeconfig
		// This is useful for local development or testing.
		var kubeconfigPath string

		// Check if KUBECONFIG is set, otherwise use the default path ~/.kube/config
		if os.Getenv("KUBECONFIG") != "" {
			kubeconfigPath = os.Getenv("KUBECONFIG")
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Fatalf("Error getting user home directory: %v", err)
			}
			kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
		}

		klog.Info("Using kubeconfig: ", kubeconfigPath)
		flag.Parse()
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			klog.Fatalf("Error building kubeconfig: %v", err)
		}
	}
	return cfg
}

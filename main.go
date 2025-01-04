package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

func main() {
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
			log.Fatalf("Error building kubeconfig: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error creating clientset: %v", err)
	}

	secrets, err := clientset.CoreV1().Secrets("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing secrets: %v", err)
	}

	for _, secret := range secrets.Items {
		if len(secret.Annotations) > 0 {
			klog.Info(secret.Name, secret.Annotations)
		}
		klog.Info(secret.Name)
	}
}

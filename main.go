package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/qjoly/randomsecret/pkg/secrets"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

const (
	operatorAnnotation = "secret.a-cup-of.coffee/enable"
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
			klog.Fatalf("Error building kubeconfig: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Info(fmt.Sprintf("Error creating clientset: %v", err))
	}

	kubeSecrets, err := clientset.CoreV1().Secrets("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		klog.Info(fmt.Sprintf("Error listing secrets: %v", err))
	}

	klog.Info(fmt.Sprintf("Found %d secrets\n", len(kubeSecrets.Items)))
	for _, secret := range kubeSecrets.Items {
		if len(secret.Annotations) > 0 {

			for key, value := range secret.Annotations {
				if key == operatorAnnotation && value == "true" {
					klog.Info(secret.Name)
					secrets.HandleSecrets(secret)
				}
			}
		}
	}
}

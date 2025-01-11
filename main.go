package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

const (
	operatorAnnotation = "secret.a-cup-of.coffee/enable"
)

func logger(msg string) {
	environmentVar := "CODER_CODE_SERVER_SERVICE_HOST" // This is a variable that is set by Coder when running in a container
	if os.Getenv(environmentVar) != "" {
		klog.Info(msg)
	} else {
		fmt.Println(msg)
	}
}

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

		logger(fmt.Sprintf("Using kubeconfig: ", kubeconfigPath))
		flag.Parse()
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			logger(fmt.Sprintf("Error building kubeconfig: %v", err))
			os.Exit(1)
		}
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger(fmt.Sprintf("Error creating clientset: %v", err))
	}

	secrets, err := clientset.CoreV1().Secrets("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger(fmt.Sprintf("Error listing secrets: %v", err))
	}

	logger(fmt.Sprintf("Found %d secrets\n", len(secrets.Items)))
	for _, secret := range secrets.Items {
		if len(secret.Annotations) > 0 {

			for key, value := range secret.Annotations {
				if key == operatorAnnotation && value == "true" {
					logger(secret.Name)
				}
			}
		}
	}
}

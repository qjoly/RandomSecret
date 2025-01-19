package kube

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	rl "k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
)

// init client
type (
	KubeClient struct {
		Clientset *kubernetes.Clientset
		Cfg       *rest.Config
		el        *leaderelection.LeaderElector
		namespace string
	}
)

const (
	lockName = "random-secret"
)

var (
	identity = fmt.Sprintf("%d", os.Getpid())
)

func NewClient() *KubeClient {
	client := &KubeClient{
		Cfg: getConfig(),
	}

	var err error

	client.Clientset, err = kubernetes.NewForConfig(client.Cfg)
	if err != nil {
		klog.Info(fmt.Sprintf("Error creating clientset: %v", err))
	}

	client.namespace = os.Getenv("NAMESPACE")
	if client.namespace == "" {
		client.namespace = "default"
	}

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

func (k *KubeClient) LeaderElection() {

	cfg, err := ctrl.GetConfig()
	if err != nil {
		panic(err.Error())
	}

	var RenewDeadline = time.Second * 5
	l, err := rl.NewFromKubeconfig(
		rl.LeasesResourceLock,
		k.namespace,
		lockName,
		rl.ResourceLockConfig{
			Identity: identity,
		},
		cfg,
		RenewDeadline,
	)
	if err != nil {
		panic(err)
	}

	el, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          l,
		LeaseDuration: time.Second * 10,
		RenewDeadline: RenewDeadline,
		RetryPeriod:   time.Second * 2,
		Name:          lockName,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) { println("I am the leader!") },
			OnStoppedLeading: func() { println("I am not the leader anymore!") },
			OnNewLeader:      func(identity string) { fmt.Printf("the leader is %s\n", identity) },
		},
	})
	if err != nil {
		panic(err)
	}

	go el.Run(context.Background())

	k.el = el

	for !k.CheckLeader() {
		fmt.Println("Waiting to become leader")
		time.Sleep(2 * time.Second)
	}

	fmt.Println("Finished leader election")
}

func (k *KubeClient) CheckLeader() bool {
	return k.el.IsLeader()
}

func (k *KubeClient) WaitForLeader() {
	for !k.el.IsLeader() {
		fmt.Println("Waiting to become leader")
		time.Sleep(2 * time.Second)
	}
}

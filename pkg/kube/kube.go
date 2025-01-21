package kube

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/qjoly/randomsecret/pkg/secrets"
	"github.com/qjoly/randomsecret/pkg/types"
	v1 "k8s.io/api/core/v1"
	kerror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	rl "k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
)

type (
	KubeClient struct {
		Clientset     *kubernetes.Clientset
		Cfg           *rest.Config
		el            *leaderelection.LeaderElector
		namespace     string
		DynamicClient dynamic.Interface
	}
)

const (
	lockName = "random-secret"
)

var (
	identity = os.Getenv("HOSTNAME")
)

func NewClient() *KubeClient {
	client := &KubeClient{
		Cfg: getConfig(),
	}

	var err error

	client.Clientset, err = kubernetes.NewForConfig(client.Cfg)
	if err != nil {
		klog.Fatal(fmt.Sprintf("Error creating clientset: %v", err))
	}

	client.namespace = os.Getenv("NAMESPACE")
	if client.namespace == "" {
		client.namespace = "default"
	}

	client.DynamicClient, err = dynamic.NewForConfig(client.Cfg)
	if err != nil {
		klog.Fatal(fmt.Sprintf("Failed to create dynamic client: %v", err))
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

	var RenewDeadline = time.Second * 10
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
		LeaseDuration: time.Second * 15,
		RenewDeadline: RenewDeadline,
		RetryPeriod:   time.Second * 5,
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
		klog.Info("Waiting to become leader")
		time.Sleep(200 * time.Millisecond)
	}

	klog.Info("Finished leader election")
}

func (k *KubeClient) CheckLeader() bool {
	return k.el.IsLeader()
}

func (k *KubeClient) WaitForLeader() {
	for !k.el.IsLeader() {
		klog.Info("Waiting to become leader")
		time.Sleep(2 * time.Second)
	}
}

func (k *KubeClient) CreateSecret(randomSecret types.RandomSecret) error {

	spec := make(map[string]string)
	spec[randomSecret.Key] = secrets.GenerateRandomSecret(int(randomSecret.Length), randomSecret.SpecialChar)

	for k, v := range randomSecret.Static {
		spec[k] = v
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      randomSecret.Name,
			Namespace: randomSecret.NameSpace,
		},
		StringData: spec,
	}

	clientset := k.Clientset

	_, err := clientset.CoreV1().Secrets(randomSecret.NameSpace).Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create secret: %v", err)
	}

	fmt.Printf("Secret %s created in namespace %s\n", randomSecret.Name, randomSecret.NameSpace)
	return nil
}

func (k *KubeClient) IsSecretCreated(secretName string, namespace string) (bool, error) {

	clientset := k.Clientset
	_, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		if kerror.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

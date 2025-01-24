package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qjoly/randomsecret/pkg/kube"
	"github.com/qjoly/randomsecret/pkg/mutating"
	"github.com/qjoly/randomsecret/pkg/randomsecrets"
	"github.com/qjoly/randomsecret/pkg/secrets"
	"github.com/qjoly/randomsecret/pkg/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

var (
	startTime = time.Now()
)

func main() {
	k := kube.NewClient()
	go mutating.Run()

	k.LeaderElection()
	clientset := k.Clientset
	dynamicClient := k.DynamicClient

	secrets.ReconcileSecrets(clientset)
	randomsecrets.ReconcileRandomSecrets(k, types.RandomSecretGVR)

	working := false

	secretWatch := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"secrets",
		metav1.NamespaceAll,
		fields.Everything(),
	)

	secretHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if !k.CheckLeader() {
				k.WaitForLeader()
			}
			secret, ok := obj.(*v1.Secret)
			if !ok {
				log.Println("Failed to cast to Secret")
				return
			}
			if secret.CreationTimestamp.Time.Before(startTime) {
				return
			}
			if secrets.IsSecretManaged(*secret) {
				working = true
				klog.Infof("Added, Found Secret %s", secret.Name)
				err := secrets.ReconcileSecrets(clientset)
				if err != nil {
					klog.Infof("Error reconciling secrets: %v", err)
				}
				working = false
			}
		},
		UpdateFunc: func(_, newObj interface{}) {
			if !k.CheckLeader() {
				k.WaitForLeader()
			}
			secret, ok := newObj.(*v1.Secret)
			if !ok {
				log.Println("Failed to cast to Secret")
				return
			}
			if secrets.IsSecretManaged(*secret) {
				working = true
				klog.Infof("Updated, Found Secret %s", secret.Name)
				err := secrets.ReconcileSecrets(clientset)
				if err != nil {
					klog.Infof("Error reconciling secrets: %v", err)
				}
				working = false
			}
		},
	}

	randomSecretWatch := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return dynamicClient.Resource(types.RandomSecretGVR).Namespace(metav1.NamespaceAll).List(context.Background(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return dynamicClient.Resource(types.RandomSecretGVR).Namespace(metav1.NamespaceAll).Watch(context.Background(), options)
		},
	}

	randomSecretHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if !k.CheckLeader() {
				k.WaitForLeader()
			}
			unstructuredObj, ok := obj.(runtime.Object)
			if !ok {
				log.Println("Failed to cast to Unstructured object")
				return
			}
			randomSecret, err := randomsecrets.ToRandomSecret(unstructuredObj)
			if err != nil {
				log.Printf("Failed to convert object to RandomSecret: %v\n", err)
				return
			}
			if randomSecret.CreationTimestamp.Before(startTime) {
				return
			}
			klog.Infof("Added, Found RandomSecret %s", randomSecret.Name)
			working = true
			err = randomsecrets.ReconcileRandomSecrets(k, types.RandomSecretGVR)
			if err != nil {
				klog.Infof("Error reconciling RandomSecrets: %v", err)
			}
			working = false
		},
		UpdateFunc: func(_, newObj interface{}) {
			if !k.CheckLeader() {
				k.WaitForLeader()
			}
			unstructuredObj, ok := newObj.(runtime.Object)
			if !ok {
				log.Println("Failed to cast to Unstructured object")
				return
			}
			randomSecret, err := randomsecrets.ToRandomSecret(unstructuredObj)
			if err != nil {
				log.Printf("Failed to convert object to RandomSecret: %v\n", err)
				return
			}
			klog.Infof("Updated, Found RandomSecret %s", randomSecret.Name)
			working = true
			err = randomsecrets.ReconcileRandomSecrets(k, types.RandomSecretGVR)
			if err != nil {
				klog.Infof("Error reconciling RandomSecrets: %v", err)
			}
			working = false
		},
	}

	secretInformer := cache.NewSharedInformer(secretWatch, &v1.Secret{}, 0)
	secretInformer.AddEventHandler(secretHandler)

	randomSecretInformer := cache.NewSharedInformer(randomSecretWatch, &unstructured.Unstructured{}, 0)
	randomSecretInformer.AddEventHandler(randomSecretHandler)

	endSignal := make(chan struct{})
	go secretInformer.Run(endSignal)
	go randomSecretInformer.Run(endSignal)
	defer close(endSignal)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	endSignal <- struct{}{}
	klog.Info("Received shutdown signal, exiting...")

	for range [10]int{} {
		if !working {
			klog.Warning("Job ended, terminating...")
			break
		}
		time.Sleep(2 * time.Second)
		klog.Warning("Waiting for job to end...")
	}
}

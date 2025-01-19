package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qjoly/randomsecret/pkg/kube"
	"github.com/qjoly/randomsecret/pkg/secrets"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

func main() {

	k := kube.NewClient()

	k.LeaderElection()
	clientset := k.Clientset
	err := reconcile(clientset)
	if err != nil {
		klog.Info(fmt.Sprintf("Error reconciling secrets: %v", err))
	}

	startTime := time.Now()

	secretWatch := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"secrets",
		metav1.NamespaceAll,
		fields.Everything(),
	)

	working := false

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

				klog.Info(fmt.Sprintf("Added, Found secret %s", secret.Name))
				err = reconcile(clientset)
				if err != nil {
					klog.Info(fmt.Sprintf("Error reconciling secrets: %v", err))
				}
				working = false
			}
		},
		UpdateFunc: func(_, newObj interface{}) {
			if !k.CheckLeader() {
				k.WaitForLeader()
			}
			// We only care about the new object
			secret, ok := newObj.(*v1.Secret)
			if !ok {
				log.Println("Failed to cast to Secret")
				return
			}

			if secrets.IsSecretManaged(*secret) {
				working = true
				klog.Info(fmt.Sprintf("Updated, Found secret %s", secret.Name))
				err = reconcile(clientset)
				if err != nil {
					klog.Info(fmt.Sprintf("Error reconciling secrets: %v", err))
				}
				working = false
			}
		},
	}

	secretInformer := cache.NewSharedInformer(secretWatch, &v1.Secret{}, 0)
	secretInformer.AddEventHandler(secretHandler)
	endSignal := make(chan struct{})
	go secretInformer.Run(endSignal)
	defer close(endSignal)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	endSignal <- struct{}{}
	fmt.Println("Received shutdown signal, exiting...")

	for range [10]int{} {
		if !working {
			klog.Warning("Job ended, terminating...")
			break
		}

		time.Sleep(2 * time.Second)
		klog.Warning("Waiting for job to end...")
	}

}

func reconcile(clientset *kubernetes.Clientset) error {

	kubeSecrets, err := clientset.CoreV1().Secrets("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		klog.Info(fmt.Sprintf("Error listing secrets: %v", err))
		return err
	}

	klog.Info(fmt.Sprintf("Found %d secrets\n", len(kubeSecrets.Items)))
	for _, secret := range kubeSecrets.Items {
		if secrets.IsSecretManaged(secret) {
			secrets.HandleSecrets(clientset, secret)
		}
	}
	return nil
}

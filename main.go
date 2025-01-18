package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/qjoly/randomsecret/pkg/kube"
	"github.com/qjoly/randomsecret/pkg/secrets"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

func main() {
	k := kube.NewClient()
	clientset := k.Clientset

	// kubeSecrets, err := clientset.CoreV1().Secrets("").List(context.Background(), metav1.ListOptions{})
	// if err != nil {
	// 	klog.Info(fmt.Sprintf("Error listing secrets: %v", err))
	// }

	// klog.Info(fmt.Sprintf("Found %d secrets\n", len(kubeSecrets.Items)))
	// for _, secret := range kubeSecrets.Items {
	// 	if len(secret.Annotations) > 0 {

	// 		if secrets.IsSecretManaged(secret) {
	// 			secrets.HandleSecrets(clientset, secret)
	// 		}
	// 	}
	// }

	secretWatch := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"secrets",
		metav1.NamespaceAll,
		fields.Everything(),
	)

	secretHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			secret, ok := obj.(*v1.Secret)
			if !ok {
				log.Println("Failed to cast to Secret")
				return
			}
			if secrets.IsSecretManaged(*secret) {
				fmt.Print("Added, ")
				fmt.Println("Found secret", secret.Name)
			}
		},
		UpdateFunc: func(_, newObj interface{}) {
			// We only care about the new object
			secret, ok := newObj.(*v1.Secret)
			if !ok {
				log.Println("Failed to cast to Secret")
				return
			}

			if secrets.IsSecretManaged(*secret) {
				fmt.Print("Updated, ")
				fmt.Println("Found secret", secret.Name)
			}
		},
	}

	secretInformer := cache.NewSharedInformer(secretWatch, &v1.Secret{}, 0)
	secretInformer.AddEventHandler(secretHandler)

	stop := make(chan struct{})
	defer close(stop)

	secretInformer.Run(stop)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	endSignal := make(chan struct{})

	go func() {
		<-c
		endSignal <- struct{}{}
	}()

	<-endSignal

}

// func reconcile() {

// 	secretWatch := cache.NewListWatchFromClient(
// 		clientset.CoreV1().RESTClient(),
// 		"secrets",
// 		metav1.NamespaceAll,
// 		fields.Everything(),
// 	)

// 	secretHandler := cache.ResourceEventHandlerFuncs{
// 		AddFunc: func(obj interface{}) {
// 			secret, ok := obj.(*v1.Secret)
// 			if !ok {
// 				log.Println("Failed to cast to Secret")
// 				return
// 			}

// 			shouldReconcile := false
// 			for key := range secret.Labels {
// 				selector := strings.Split(users.SecretUserSelector, "=")
// 				if key == selector[0] {
// 					klog.Infof("Secret %s has been added", secret.Name)
// 					shouldReconcile = true
// 					break
// 				}
// 			}
// 			if !shouldReconcile {
// 				return
// 			}

// 			if runningRoutine {
// 				klog.Info("Already running a routine, skipping")
// 				return
// 			}
// 			runningRoutine = true
// 			err := reconcileMinio(minioCredentials)
// 			if err != nil {
// 				klog.Errorf("Error handling Secret: %s", err.Error())
// 				os.Exit(1)
// 			}
// 			runningRoutine = false
// 		},
// 		UpdateFunc: func(_, newObj interface{}) {
// 			secret, ok := newObj.(*v1.Secret)
// 			if !ok {
// 				log.Println("Failed to cast to Secret")
// 				return
// 			}

// 			shouldReconcile := false
// 			for key := range secret.Labels {
// 				selector := strings.Split(users.SecretUserSelector, "=")
// 				if key == selector[0] {
// 					klog.Infof("Secret %s has been updated", secret.Name)
// 					shouldReconcile = true
// 					break
// 				}
// 			}
// 			if !shouldReconcile {
// 				return
// 			}

// 			if runningRoutine {
// 				klog.Info("Already running a routine, skipping")
// 				return
// 			}
// 			runningRoutine = true
// 			err := reconcileMinio(minioCredentials)
// 			if err != nil {
// 				klog.Errorf("Error handling Secret: %s", err.Error())
// 				os.Exit(1)
// 			}
// 			runningRoutine = false
// 		},
// 		DeleteFunc: func(obj interface{}) {
// 			secret, ok := obj.(*v1.Secret)
// 			if !ok {
// 				log.Println("Failed to cast to Secret")
// 				return
// 			}

// 			shouldReconcile := false
// 			for key := range secret.Labels {
// 				selector := strings.Split(users.SecretUserSelector, "=")
// 				if key == selector[0] {
// 					klog.Infof("Secret %s has been deleted", secret.Name)
// 					shouldReconcile = true
// 					break
// 				}
// 			}
// 			if !shouldReconcile {
// 				return
// 			}
// 			if runningRoutine {
// 				klog.Info("Already running a routine, skipping")
// 				return
// 			}
// 			runningRoutine = true
// 			err := reconcileMinio(minioCredentials)
// 			if err != nil {
// 				klog.Errorf("Error handling Secret: %s", err.Error())
// 				os.Exit(1)
// 			}
// 			runningRoutine = false
// 		},
// 	}
// }

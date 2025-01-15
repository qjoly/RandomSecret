package secrets

import (
	"context"
	"fmt"

	"github.com/qjoly/randomsecret/pkg/types"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// getRandomSecretKey returns the key used to store the random secret
// Check if the annotations map contains the key annotation
// If it does, return the key
func getRandomSecretKey(secret v1.Secret) string {
	key, ok := secret.Annotations[types.OperatorSecretKeyAnnotation]
	if !ok {
		klog.Info("Secret does not have a key annotation, using default key")
		return types.OperatorDefaultSecretKey
	}
	klog.Info("Secret has a key annotation, using key: ", key)
	return key
}

func IsSecretManaged(secret v1.Secret) bool {
	value, ok := secret.Annotations[types.OperatorEnabledAnnotation]
	if !ok {
		return false
	}

	if value != "true" {
		return false
	}

	return ok
}

func isAlreadyHandled(secret v1.Secret) bool {
	_, ok := secret.Data[getRandomSecretKey(secret)]
	if !ok {
		return false
	}
	return true
}

func HandleSecrets(clientset *kubernetes.Clientset, secret v1.Secret) {
	klog.Info(fmt.Sprintf("Handling secret: %s with key %s", secret.Name, getRandomSecretKey(secret)))

	if isAlreadyHandled(secret) {
		klog.Info(fmt.Sprintf("Secret %s already handled", secret.Name))
		return
	}
	randomPass := generateRandomSecret("password")
	patchSecret(clientset, secret, getRandomSecretKey(secret), randomPass)
}

func generateRandomSecret(pattern string) string {
	// Generate a random secret
	// For now, we will just return the pattern
	// In the future, we will implement the random generation
	return pattern
}

// patchSecret updates the secret with the new value
func patchSecret(clientset *kubernetes.Clientset, secret v1.Secret, key string, value string) {
	secret.Data[key] = []byte(value)
	_, err := clientset.CoreV1().Secrets(secret.Namespace).Update(context.Background(), &secret, metav1.UpdateOptions{})
	if err != nil {
		klog.Info(fmt.Sprintf("Error patching secret %s: %v", secret.Name, err))
	}

}

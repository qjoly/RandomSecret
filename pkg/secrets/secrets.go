package secrets

import (
	"context"
	"fmt"
	"time"

	"github.com/qjoly/randomsecret/pkg/types"
	"golang.org/x/exp/rand"
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
	randomPass := generateRandomSecret(secret.Annotations)
	patchSecret(clientset, secret, getRandomSecretKey(secret), randomPass)
}

func generateRandomSecret(annotations map[string]string) string {

	klog.Info("Generating random secret")
	// Check if the annotation is nil or empty
	pattern := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// Check if the special char annotation is present and different from false
	// Then add special characters to the pattern otherwise use the default pattern
	if annotations[types.OperatorSpecialCharAnnotation] != "" || annotations[types.OperatorSpecialCharAnnotation] != "false" {
		pattern += "!@#$%^&*()_+"
	}

	var length int

	// Check if the length annotation is present
	// If it is, convert the value to an integer
	if annotations[types.OperatorLengthAnnotation] != "" {
		fmt.Sscanf(annotations[types.OperatorLengthAnnotation], "%d", &length)
	}

	rand.Seed(uint64(time.Now().UnixNano()))

	secret := make([]byte, length)
	for i := range secret {
		secret[i] = pattern[rand.Intn(len(pattern))]
	}

	return string(secret)
}

// patchSecret updates the secret with the new value
func patchSecret(clientset *kubernetes.Clientset, secret v1.Secret, key string, value string) {

	// If the secret data map is nil, create a new map
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}

	secret.Data[key] = []byte(value)

	_, err := clientset.CoreV1().Secrets(secret.Namespace).Update(context.Background(), &secret, metav1.UpdateOptions{})
	if err != nil {
		klog.Info(fmt.Sprintf("Error patching secret %s: %v", secret.Name, err))
	}

	klog.Info(fmt.Sprintf("Secret %s patched with key %s", secret.Name, key))

}

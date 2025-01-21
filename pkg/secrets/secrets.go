package secrets

import (
	"context"
	"fmt"
	"strconv"
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
	return ok
}

func HandleSecrets(clientset *kubernetes.Clientset, secret v1.Secret) {
	klog.Info(fmt.Sprintf("Handling secret: %s with key %s", secret.Name, getRandomSecretKey(secret)))

	if isAlreadyHandled(secret) {
		klog.Info(fmt.Sprintf("Secret %s already handled", secret.Name))
		return
	}

	newSecret, err := MutateSecret(secret)
	if err != nil {
		klog.Info(fmt.Sprintf("Error mutating secret %s: %v", secret.Name, err))
		return
	}

	patchSecret(clientset, newSecret)
}

func GenerateRandomSecret(length int, specialChar bool) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const special = "!@#$%^&*()-_=+[]{}|;:,.<>?/~`"

	rand.Seed(uint64(time.Now().UnixNano()))

	charset := letters
	if specialChar {
		charset += special
	}

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}

	return string(b)
}

// patchSecret updates the secret with the new value
func patchSecret(clientset *kubernetes.Clientset, secret v1.Secret) {

	_, err := clientset.CoreV1().Secrets(secret.Namespace).Update(context.Background(), &secret, metav1.UpdateOptions{})
	if err != nil {
		klog.Info(fmt.Sprintf("Error patching secret %s: %v", secret.Name, err))
	}

	klog.Info(fmt.Sprintf("Secret %s patched", secret.Name))

}

func ReconcileSecrets(clientset *kubernetes.Clientset) error {
	secretsList, err := clientset.CoreV1().Secrets("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		klog.Infof("Error listing secrets: %v", err)
		return err
	}
	klog.Infof("Found %d secrets\n", len(secretsList.Items))
	for _, secret := range secretsList.Items {
		if IsSecretManaged(secret) {
			HandleSecrets(clientset, secret)
		}
	}
	return nil
}

func MutateSecret(secret v1.Secret) (v1.Secret, error) {
	secretCopy := secret.DeepCopy()
	var length int

	if secretCopy.Annotations[types.OperatorLengthAnnotation] == "" {
		length = 32
	} else {
		var err error
		length, err = strconv.Atoi(secretCopy.Annotations[types.OperatorLengthAnnotation])
		if err != nil {
			klog.Infof("Invalid length annotation, using default length 32: %v", err)
			length = 32
		}
	}

	var specialChar bool

	if secret.Annotations[types.OperatorSpecialCharAnnotation] == "" {
		specialChar = true
	} else {
		var err error
		specialChar, err = strconv.ParseBool(secret.Annotations[types.OperatorSpecialCharAnnotation])
		if err != nil {
			klog.Infof("Invalid specialChar annotation, using default specialChar true: %v", err)
			specialChar = true
			return v1.Secret{}, err
		}
	}

	randomPass := GenerateRandomSecret(length, specialChar)
	secretCopy.Data[getRandomSecretKey(secret)] = []byte(randomPass)

	return *secretCopy, nil
}

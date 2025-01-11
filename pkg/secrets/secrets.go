package secrets

import (
	"fmt"

	"github.com/qjoly/randomsecret/pkg/types"
	v1 "k8s.io/api/core/v1"
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

func HandleSecrets(secret v1.Secret) {
	// Do something with the secret
	// For example, print the secret name
	fmt.Println(secret.Name)

}

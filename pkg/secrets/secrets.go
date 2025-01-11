package secrets

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

func HandleSecrets(secret v1.Secret) {
	// Do something with the secret
	// For example, print the secret name
	fmt.Println(secret.Name)

}

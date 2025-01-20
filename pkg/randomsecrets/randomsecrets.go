package randomsecrets

import (
	"context"
	"time"

	"github.com/qjoly/randomsecret/pkg/kube"
	"github.com/qjoly/randomsecret/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog"
)

func ToRandomSecret(unstructuredObj runtime.Object) (types.RandomSecret, error) {

	// On aurait pu utiliser un yaml.Unmarshal pour parser l'objet unstructuredObj
	// mais on a choisi de le faire à la main pour plus de clarté
	var randomSecret types.RandomSecret

	objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(unstructuredObj)
	if err != nil {
		return randomSecret, err
	}
	creationTimestampStr := objMap["metadata"].(map[string]interface{})["creationTimestamp"].(string) // e.g. 2025-01-19T17:55:32Z
	creationTimestamp, err := time.Parse(time.RFC3339, creationTimestampStr)
	if err != nil {
		return randomSecret, err
	}

	name := objMap["metadata"].(map[string]interface{})["name"].(string)
	specMap := objMap["spec"].(map[string]interface{})
	length := specMap["length"].(int64)

	var specialChar bool
	if specMap["specialChar"] == nil {
		specialChar = true
	} else {
		specialChar = specMap["specialChar"].(bool)
	}
	key := specMap["key"].(string)
	secretName := specMap["secretName"].(string)

	var static map[string]string
	if specMap["static"] != nil {
		static = specMap["static"].(map[string]string)
	}

	NameSpace := objMap["metadata"].(map[string]interface{})["namespace"].(string)

	randomSecret = types.RandomSecret{
		Name:              name,
		Length:            length,
		SpecialChar:       specialChar,
		Key:               key,
		SecretName:        secretName,
		Static:            static,
		CreationTimestamp: creationTimestamp,
		NameSpace:         NameSpace,
	}

	return randomSecret, nil
}

func ReconcileRandomSecrets(k *kube.KubeClient, randomSecretGVR schema.GroupVersionResource) error {

	dynamicClient := k.DynamicClient
	randomSecretsList, err := dynamicClient.Resource(randomSecretGVR).Namespace("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		klog.Infof("Error listing RandomSecrets: %v", err)
		return err
	}
	klog.Infof("Found %d RandomSecrets\n", len(randomSecretsList.Items))
	for _, item := range randomSecretsList.Items {
		randomSecret, err := ToRandomSecret(&item)
		if err != nil {
			klog.Infof("Failed to parse RandomSecret: %v", err)
			continue
		}

		isSecretAlreadyCreated, err := k.IsSecretCreated(randomSecret.Name, randomSecret.NameSpace)
		if err != nil {
			klog.Infof("Failed to check if secret %s is already created: %v", randomSecret.Name, err)
			continue
		}
		if !isSecretAlreadyCreated {
			err = k.CreateSecret(randomSecret)
			if err != nil {
				klog.Infof("Failed to create secret %s: %v", randomSecret.Name, err)
				updateStatusReady(randomSecret, k, false)
			}

			updateStatusReady(randomSecret, k, true)
		} else {
			klog.Infof("Secret %s already exists", randomSecret.Name)
		}

	}
	return nil
}

func updateStatusReady(randomSecret types.RandomSecret, k *kube.KubeClient, ready bool) {
	dynamicClient := k.DynamicClient
	randomSecretGVR := types.RandomSecretGVR

	randomSecretObj, err := dynamicClient.Resource(randomSecretGVR).Namespace(randomSecret.NameSpace).Get(context.Background(), randomSecret.Name, metav1.GetOptions{})
	if err != nil {
		klog.Infof("Failed to get RandomSecret %s: %v", randomSecret.Name, err)
		return
	}

	readyString := "NotReady"
	if ready {
		readyString = "Ready"
	}
	randomSecretObj.Object["status"] = map[string]interface{}{
		"state": readyString,
	}
	_, err = dynamicClient.Resource(randomSecretGVR).Namespace(randomSecret.NameSpace).Update(context.Background(), randomSecretObj, metav1.UpdateOptions{})
	if err != nil {
		klog.Infof("Failed to update status of RandomSecret %s: %v", randomSecret.Name, err)
	}
}

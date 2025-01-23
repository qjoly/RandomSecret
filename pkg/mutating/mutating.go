package mutating

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/qjoly/randomsecret/pkg/secrets"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	scheme = runtime.NewScheme()
	//codecs       = serializer.NewCodecFactory(scheme)
	//deserializer = codecs.UniversalDeserializer()
)

func init() {
	_ = admissionv1.AddToScheme(scheme)
}

func Run() {
	http.HandleFunc("/mutate", func(w http.ResponseWriter, r *http.Request) {
		var admissionReview admissionv1.AdmissionReview
		if err := json.NewDecoder(r.Body).Decode(&admissionReview); err != nil {
			http.Error(w, fmt.Sprintf("Error decoding request: %v", err), http.StatusBadRequest)
			return
		}

		klog.Infof("Received request: %s in namespace %s", admissionReview.Request.Name, admissionReview.Request.Namespace)

		decoder := admission.NewDecoder(runtime.NewScheme())

		var patchBytes []byte
		admissionResponse := admissionv1.AdmissionResponse{
			UID:     admissionReview.Request.UID,
			Allowed: true,
		}

		secret := &corev1.Secret{}
		if err := decoder.DecodeRaw(admissionReview.Request.Object, secret); err != nil {
			http.Error(w, fmt.Sprintf("Error decoding secret: %v", err), http.StatusBadRequest)
			return
		}

		if !secrets.IsSecretManaged(*secret) {
			klog.Infof("Secret %s is not managed, returning", secret.Name)
			patchBytes, err := json.Marshal(secret)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error marshalling patch: %v", err), http.StatusInternalServerError)
				return
			}

			admissionResponse.Patch = patchBytes
			admissionResponse.PatchType = nil
			admissionReview.Response = &admissionResponse

			fmt.Println(admissionReview.Response)
			if err := json.NewEncoder(w).Encode(admissionReview); err != nil {
				http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
				return
			}

			return
		}

		patch, err := secrets.MutateSecret(*secret)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error mutating secret: %v", err), http.StatusInternalServerError)
			return
		}

		patchBytes, err = json.Marshal(patch)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error marshalling patch: %v", err), http.StatusInternalServerError)
			return
		}

		admissionResponse.Patch = patchBytes

		admissionReview.Response = &admissionResponse
		if err := json.NewEncoder(w).Encode(admissionReview); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
			return
		}
	})

	klog.Info("Starting webhook server on :443")
	if err := http.ListenAndServeTLS(":443", "/certs/tls.crt", "/certs/tls.key", nil); err != nil {
		klog.Errorf("Failed to start server: %v\n", err)
	}
}

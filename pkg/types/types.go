package types

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	OperatorPrefixAnnotation      = "secret.a-cup-of.coffee/"
	OperatorEnabledAnnotation     = OperatorPrefixAnnotation + "enable"
	OperatorSecretKeyAnnotation   = OperatorPrefixAnnotation + "key"
	OperatorDefaultSecretKey      = "password"
	OperatorSpecialCharAnnotation = OperatorPrefixAnnotation + "special-char"
	OperatorLengthAnnotation      = OperatorPrefixAnnotation + "length"
)

var (
	RandomSecretGVR = schema.GroupVersionResource{
		Group:    "secret.a-cup-of.coffee",
		Version:  "v1",
		Resource: "randomsecrets",
	}
)

type RandomSecret struct {
	Name              string
	Length            int64
	SpecialChar       bool
	Key               string
	SecretName        string
	Static            map[string]string
	CreationTimestamp time.Time
	NameSpace         string
}

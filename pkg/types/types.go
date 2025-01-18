package types

const (
	OperatorPrefixAnnotation      = "secret.a-cup-of.coffee/"
	OperatorEnabledAnnotation     = OperatorPrefixAnnotation + "enable"
	OperatorSecretKeyAnnotation   = OperatorPrefixAnnotation + "key"
	OperatorDefaultSecretKey      = "password"
	OperatorSpecialCharAnnotation = OperatorPrefixAnnotation + "special-char"
	OperatorLengthAnnotation      = OperatorPrefixAnnotation + "length"
)

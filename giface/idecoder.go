package giface

type IDecoder interface {
	IInterceptor
	GetLengthField() *LengthField
}

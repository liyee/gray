package giface

type IcReq interface{} //拦截器输入数据

type IcResp interface{} //拦截器输出数据

type IInterceptor interface {
	Intercept(IChain) IcResp
}

type IChain interface {
	Request() IcReq
	GetIMessage() IMessage
	Proceed(IcReq) IcResp
	ProceedWithIMessage(IMessage, IcReq) IcResp
}

package giface

type IMsgHandler interface {
	AddRouter(msgID uint32, router IRouter)
	AddRouterSlices(msgID uint32, hander ...RouterHandler) IRouterSlices
	Group(start, end uint32, handers ...RouterHandler) IGroupRouterSlices
	Use(handers ...RouterHandler) IRouterSlices

	StartWorkerPool()
	SendMsgToTaskQueue(request IRequest)

	Execute(request IRequest)

	AddInterceptor(interceptor IInterceptor)
	SetHeadInterceptor(interceptor IInterceptor)
}

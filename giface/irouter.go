package giface

type RouterHandler func(request IRequest)

type IRouter interface {
	PreHandle(request IRequest)
	Handle(request IRequest)
	PostHandle(request IRequest)
}

type IRouterSlices interface {
	Use(Handlers ...RouterHandler)
	AddHandler(msgID uint32, handlers ...RouterHandler)
	Group(start, end uint32, handlers ...RouterHandler) IGroupRouterSlices
	GetHandlers(msgID uint32) ([]RouterHandler, bool)
}

type IGroupRouterSlices interface {
	Use(handlers ...RouterHandler)
	AddHandler(msgID uint32, handlers ...RouterHandler)
}

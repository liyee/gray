package gnet

import (
	"strconv"
	"sync"

	"github.com/liyee/gtcp/giface"
)

type BaseRouter struct{}

func (br *BaseRouter) PreHandle(request giface.IRequest)  {}
func (br *BaseRouter) Handle(request giface.IRequest)     {}
func (br *BaseRouter) PostHandle(request giface.IRequest) {}

type RouterSlices struct {
	Apis     map[uint32][]giface.RouterHandler
	Handlers []giface.RouterHandler
	sync.RWMutex
}

func NewRouterSlices() *RouterSlices {
	return &RouterSlices{
		Apis:     make(map[uint32][]giface.RouterHandler, 10),
		Handlers: make([]giface.RouterHandler, 0, 6),
	}
}

func (r *RouterSlices) Use(handlers ...giface.RouterHandler) {
	r.Handlers = append(r.Handlers, handlers...)
}

func (r *RouterSlices) AddHandler(msgID uint32, handlers ...giface.RouterHandler) {
	if _, ok := r.Apis[msgID]; ok {
		panic("repead api, msgID =" + strconv.Itoa(int(msgID)))
	}

	finalSize := len(r.Handlers) + len(handlers)
	mergeHandlers := make([]giface.RouterHandler, finalSize)
	copy(mergeHandlers, r.Handlers)
	copy(mergeHandlers[len(r.Handlers):], handlers)
	r.Apis[msgID] = append(r.Apis[msgID], mergeHandlers...)
}

func (r *RouterSlices) GetHandlers(msgID uint32) ([]giface.RouterHandler, bool) {
	r.RLock()
	defer r.RUnlock()
	handlers, ok := r.Apis[msgID]
	return handlers, ok
}

func (r *RouterSlices) Group(start, end uint32, handlers ...giface.RouterHandler) giface.IGroupRouterSlices {
	return NewGroup(start, end, r, handlers...)
}

type GroupRouter struct {
	start    uint32
	end      uint32
	Handlers []giface.RouterHandler
	router   giface.IRouterSlices
}

func NewGroup(start, end uint32, router *RouterSlices, handlers ...giface.RouterHandler) *GroupRouter {
	g := &GroupRouter{
		start:    start,
		end:      end,
		Handlers: make([]giface.RouterHandler, 0, len(handlers)),
		router:   router,
	}
	g.Handlers = append(g.Handlers, handlers...)

	return g
}

func (g *GroupRouter) Use(handlers ...giface.RouterHandler) {
	g.Handlers = append(g.Handlers, handlers...)
}

func (g *GroupRouter) AddHandler(msgID uint32, handlers ...giface.RouterHandler) {
	if msgID < g.start || msgID > g.end {
		panic("add router to goup err in msgID:" + strconv.Itoa(int(msgID)))
	}

	finalSize := len(g.Handlers) + len(handlers)
	mergeHandlers := make([]giface.RouterHandler, finalSize)
	copy(mergeHandlers, g.Handlers)
	copy(mergeHandlers[len(g.Handlers):], handlers)

	g.router.AddHandler(msgID, mergeHandlers...)
}

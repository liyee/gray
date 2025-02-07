package gnet

import (
	"github.com/liyee/gtcp/giface"
	"github.com/liyee/gtcp/ginterceptor"
)

type chainBuilder struct {
	body       []giface.IInterceptor
	head, tail giface.IInterceptor
}

func newChainBuilder() *chainBuilder {
	return &chainBuilder{
		body: make([]giface.IInterceptor, 0),
	}
}
func (ic *chainBuilder) Head(interceptor giface.IInterceptor) {
	ic.head = interceptor
}

func (ic *chainBuilder) Tail(interceptor giface.IInterceptor) {
	ic.tail = interceptor
}
func (ic *chainBuilder) AddInterceptor(interceptor giface.IInterceptor) {
	ic.body = append(ic.body, interceptor)
}

func (ic *chainBuilder) Execute(req giface.IcReq) giface.IcResp {

	// Put all the interceptors into the builder
	var interceptors []giface.IInterceptor
	if ic.head != nil {
		interceptors = append(interceptors, ic.head)
	}
	if len(ic.body) > 0 {
		interceptors = append(interceptors, ic.body...)
	}
	if ic.tail != nil {
		interceptors = append(interceptors, ic.tail)
	}

	// Create a new interceptor chain and execute each interceptor
	chain := ginterceptor.NewChain(interceptors, 0, req)

	// Execute the chain
	return chain.Proceed(req)
}

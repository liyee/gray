package ginterceptor

import (
	"github.com/liyee/gtcp/giface"
)

type Chain struct {
	req         giface.IcReq
	position    int
	interceptor []giface.IInterceptor
}

func NewChain(list []giface.IInterceptor, pos int, req giface.IcReq) giface.IChain {
	return &Chain{
		req:         req,
		position:    pos,
		interceptor: list,
	}
}

func (c *Chain) ShouldIRequest(icReq giface.IcReq) giface.IRequest {
	if icReq == nil {
		return nil
	}

	switch icReq.(type) {
	case giface.IRequest:
		return icReq.(giface.IRequest)
	default:
		return nil
	}
}

func (c *Chain) Request() giface.IcReq {
	return c.req
}
func (c *Chain) GetIMessage() giface.IMessage {
	req := c.Request()
	if req == nil {
		return nil
	}

	iRequest := c.ShouldIRequest(req)
	if iRequest == nil {
		return nil
	}

	return iRequest.GetMessage()
}
func (c *Chain) Proceed(request giface.IcReq) giface.IcResp {
	if c.position < len(c.interceptor) {
		chain := NewChain(c.interceptor, c.position+1, request)
		interceptor := c.interceptor[c.position]
		response := interceptor.Intercept(chain)
		return response
	}
	return request
}

func (c *Chain) ProceedWithIMessage(iMessage giface.IMessage, response giface.IcReq) giface.IcResp {
	if iMessage == nil || response == nil {
		return c.Proceed(c.Request())
	}

	req := c.Request()
	if req == nil {
		return c.Proceed(c.Request())
	}

	iRequest := c.ShouldIRequest(req)
	if iRequest == nil {
		return c.Proceed(c.Request())
	}

	iRequest.SetResPonse(response)

	return c.Proceed(iRequest)
}

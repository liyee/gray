package gnet

import (
	"math"
	"sync"

	"github.com/liyee/gray/gconf"
	"github.com/liyee/gray/giface"
	"github.com/liyee/gray/gpack"
)

const (
	PRE_HANDLE  giface.HandleStep = iota // PreHandle for pre-processing
	HANDLE                               // Handle for processing
	POST_HANDLE                          // PostHandle for post-processing

	HANDLE_OVER
)

var RequestPool = new(sync.Pool)

func init() {
	RequestPool.New = func() interface{} {
		return allocateRequest()
	}
}

type Request struct {
	giface.BaseRequest
	conn     giface.IConnection     // the connection which has been established with the client(已经和客户端建立好的链接)
	msg      giface.IMessage        // the request data sent by the client(客户端请求的数据)
	router   giface.IRouter         // the router that handles this request(请求处理的函数)
	steps    giface.HandleStep      // used to control the execution of router functions(用来控制路由函数执行)
	stepLock sync.RWMutex           // concurrency lock(并发互斥)
	needNext bool                   // whether to execute the next router function(是否需要执行下一个路由函数)
	icResp   giface.IcResp          // response data returned by the interceptors (拦截器返回数据)
	handlers []giface.RouterHandler // router function slice(路由函数切片)
	index    int8                   // router function slice index(路由函数切片索引)
	keys     map[string]interface{} // keys 路由处理时可能会存取的上下文信息
}

func (r *Request) GetResPonse() giface.IcResp {
	return r.icResp
}

func (r *Request) SetResPonse(response giface.IcResp) {
	r.icResp = response
}

func NewRequest(conn giface.IConnection, msg giface.IMessage) giface.IRequest {
	req := new(Request)
	req.steps = PRE_HANDLE
	req.conn = conn
	req.msg = msg
	req.stepLock = sync.RWMutex{}
	req.needNext = true
	req.index = -1
	return req
}

func GetRequest(conn giface.IConnection, msg giface.IMessage) giface.IRequest {
	// 根据当前模式判断是否使用对象池
	if gconf.GlobalObject.RequestPoolMode {
		// 从对象池中取得一个 Request 对象,如果池子中没有可用的 Request 对象则会调用 allocateRequest 函数构造一个新的对象分配
		r := RequestPool.Get().(*Request)
		// 因为取出的 Request 对象可能是已存在也可能是新构造的,无论是哪种情况都应该初始化再返回使用
		r.Reset(conn, msg)
		return r
	}
	return NewRequest(conn, msg)
}

func PutRequest(request giface.IRequest) {
	// 判断是否开启了对象池模式
	if gconf.GlobalObject.RequestPoolMode {
		RequestPool.Put(request)
	}
}

func allocateRequest() giface.IRequest {
	req := new(Request)
	req.steps = PRE_HANDLE
	req.needNext = true
	req.index = -1
	return req
}

func (r *Request) Reset(conn giface.IConnection, msg giface.IMessage) {
	r.steps = PRE_HANDLE
	r.conn = conn
	r.msg = msg
	r.needNext = true
	r.index = -1
	r.keys = nil
}

func (r *Request) Copy() giface.IRequest {
	// 构造一个新的 Request 对象，复制部分原始对象的参数,但是复制的 Request 不应该再对原始连接操作,所以不含有连接参数
	// 同理也不应该再执行路由方法,路由函数也不包含
	newRequest := &Request{
		conn:     nil,
		router:   nil,
		steps:    r.steps,
		needNext: false,
		icResp:   nil,
		handlers: nil,
		index:    math.MaxInt8,
	}

	// 复制原本的上下文信息
	newRequest.keys = make(map[string]interface{})
	for k, v := range r.keys {
		newRequest.keys[k] = v
	}

	// 复制一份原本的 icResp
	copyResp := []giface.IcResp{r.icResp}
	newIcResp := make([]giface.IcResp, 0, 1)
	copy(newIcResp, copyResp)
	for _, v := range newIcResp {
		newRequest.icResp = v
	}
	// 复制一份原本的 msg 信息
	newRequest.msg = gpack.NewMessageByMsgID(r.msg.GetMsgID(), r.msg.GetDataLen(), r.msg.GetRawData())

	return newRequest
}

// Set 在 Request 中存放一个上下文，如果 keys 为空会实例化一个
func (r *Request) Set(key string, value interface{}) {
	r.stepLock.Lock()
	if r.keys == nil {
		r.keys = make(map[string]interface{})
	}

	r.keys[key] = value
	r.stepLock.Unlock()
}

// Get 在 Request 中取出一个上下文信息
func (r *Request) Get(key string) (value interface{}, exists bool) {
	r.stepLock.RLock()
	value, exists = r.keys[key]
	r.stepLock.RUnlock()
	return
}

func (r *Request) GetMessage() giface.IMessage {
	return r.msg
}

func (r *Request) GetConnection() giface.IConnection {
	return r.conn
}

func (r *Request) GetData() []byte {
	return r.msg.GetData()
}

func (r *Request) GetMsgID() uint32 {
	return r.msg.GetMsgID()
}

func (r *Request) BindRouter(router giface.IRouter) {
	r.router = router
}

func (r *Request) next() {
	if r.needNext == false {
		r.needNext = true
		return
	}

	r.stepLock.Lock()
	r.steps++
	r.stepLock.Unlock()
}

func (r *Request) Goto(step giface.HandleStep) {
	r.stepLock.Lock()
	r.steps = step
	r.needNext = false
	r.stepLock.Unlock()
}

func (r *Request) Call() {

	if r.router == nil {
		return
	}

	for r.steps < HANDLE_OVER {
		switch r.steps {
		case PRE_HANDLE:
			r.router.PreHandle(r)
		case HANDLE:
			r.router.Handle(r)
		case POST_HANDLE:
			r.router.PostHandle(r)
		}

		r.next()
	}

	r.steps = PRE_HANDLE
}

func (r *Request) Abort() {
	if gconf.GlobalObject.RouterSlicesMode {
		r.index = int8(len(r.handlers))
	} else {
		r.stepLock.Lock()
		r.steps = HANDLE_OVER
		r.stepLock.Unlock()
	}
}

// BindRouterSlices New version
func (r *Request) BindRouterSlices(handlers []giface.RouterHandler) {
	r.handlers = handlers
}

func (r *Request) RouterSlicesNext() {
	r.index++
	for r.index < int8(len(r.handlers)) {
		r.handlers[r.index](r)
		r.index++
	}
}

package giface

type HandleStep int

type IFuncRequest interface {
	CallFunc()
}

type IRequest interface {
	GetConnection() IConnection

	GetData() []byte
	GetMsgID() uint32

	GetMessage() IMessage

	GetResPonse() IcResp
	SetResPonse(IcResp)

	BindRouter(router IRouter)

	Call()
	Abort()
	Goto(HandleStep)

	BindRouterSlices([]RouterHandler)
	RouterSlicesNext()

	Copy() IRequest
	Set(key string, value interface{})
	Get(key string) (value interface{}, exists bool)
}

type BaseRequest struct{}

func (br *BaseRequest) GetConnection() IConnection       { return nil }
func (br *BaseRequest) GetData() []byte                  { return nil }
func (br *BaseRequest) GetMsgID() uint32                 { return 0 }
func (br *BaseRequest) GetMessage() IMessage             { return nil }
func (br *BaseRequest) GetResponse() IcResp              { return nil }
func (br *BaseRequest) SetResponse(resp IcResp)          {}
func (br *BaseRequest) BindRouter(router IRouter)        {}
func (br *BaseRequest) Call()                            {}
func (br *BaseRequest) Abort()                           {}
func (br *BaseRequest) Goto(HandleStep)                  {}
func (br *BaseRequest) BindRouterSlices([]RouterHandler) {}
func (br *BaseRequest) RouterSlicesNext()                {}
func (br *BaseRequest) Copy() IRequest                   { return nil }

func (br *BaseRequest) Set(key string, value interface{}) {}

func (br *BaseRequest) Get(key string) (value interface{}, exists bool) { return nil, false }

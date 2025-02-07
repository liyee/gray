package giface

type IHeartbeatChecker interface {
	SetOnRemoteNotAlive(OnRemoteNotAlive)
	SetHeartbeatMsgFunc(HeartBeatMsgFunc)
	SetHeartbeatFunc(HeartBeatFunc)
	BindRouter(uint32, IRouter)
	BindRouterSlices(uint32, ...RouterHandler)
	Start()
	Stop()
	SendHeartBeatMsg() error
	BindConn(IConnection)
	Clone() IHeartbeatChecker
	MsgID() uint32
	Router() IRouter
	RouterSlices() []RouterHandler
}

type HeartBeatMsgFunc func(IConnection) []byte //用户自定义的心跳检测消息处理方法
type HeartBeatFunc func(IConnection) error     //用户自定义心跳函数
type OnRemoteNotAlive func(IConnection)        //用户自定义的远程连接不存活时的处理方法

type HeartBeatOption struct {
	MakeMsg          HeartBeatMsgFunc // User-defined method for handling heartbeat detection messages(用户自定义的心跳检测消息处理方法)
	OnRemoteNotAlive OnRemoteNotAlive // User-defined method for handling remote connections that are not alive(用户自定义的远程连接不存活时的处理方法)
	HeartBeatMsgID   uint32           // User-defined ID for heartbeat detection messages(用户自定义的心跳检测消息ID)
	Router           IRouter          // User-defined business processing route for heartbeat detection messages(用户自定义的心跳检测消息业务处理路由)
	RouterSlices     []RouterHandler  //新版本的路由处理函数的集合
}

const (
	HeartBeatDefaultMsgID uint32 = 99999
)

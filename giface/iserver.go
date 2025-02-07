package giface

import (
	"net/http"
	"time"
)

type IServer interface {
	Start()
	Stop()
	Serve()

	AddRouter(msgID uint32, router IRouter)
	AddRouterSlices(msgID uint32, handlers ...RouterHandler) IRouterSlices
	Group(start, end uint32, handlers ...RouterHandler) IGroupRouterSlices
	Use(handlers ...RouterHandler) IRouterSlices

	GetConnMgr() IConnManager //得到链接管理

	SetOnConnStart(func(IConnection))  //设置该Server的连接创建时Hook函数
	SetOnConnStop(func(IConnection))   //设置该Server的连接断开时的Hook函数
	GetOnConnStart() func(IConnection) //得到该Server的连接创建时Hook函数

	GetOnConnStop() func(IConnection) //得到该Server的连接断开时的Hook函数

	GetPacket() IDataPack       //获取Server绑定的数据协议封包方式
	GetMsgHandler() IMsgHandler //获取Server绑定的消息处理模块

	SetPacket(IDataPack) //设置Server绑定的数据协议封包方式

	StartHeartBeat(time.Duration)                             //启动心跳检测
	StartHeartBeatWithOption(time.Duration, *HeartBeatOption) //启动心跳检测(自定义回调)
	GetHeartBeat() IHeartbeatChecker                          //获取心跳检测器

	GetLengthField() *LengthField
	SetDecoder(IDecoder)
	AddInterceptor(IInterceptor)

	// Add WebSocket authentication method
	// (添加websocket认证方法)
	SetWebsocketAuth(func(r *http.Request) error)

	// Get the server name (获取服务器名称)
	ServerName() string
}

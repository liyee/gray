package gnet

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/liyee/gray/gconf"
	"github.com/liyee/gray/giface"
	"github.com/liyee/gray/ginterceptor"
	"github.com/liyee/gray/glog"
	"github.com/liyee/gray/gpack"

	"github.com/gorilla/websocket"
)

type Connection struct {
	conn       net.Conn
	bufWriter  *bufio.Writer
	connID     uint64
	connIdStr  string
	workerID   uint32
	msgHandler giface.IMsgHandler

	ctx    context.Context
	cancel context.CancelFunc

	// Buffered channel used for message communication between the read and write goroutines
	// (有缓冲管道，用于读、写两个goroutine之间的消息通信)
	msgBuffChan chan []byte

	// Go StartWriter Flag
	// (开始初始化写协程标志)
	startWriterFlag int32

	// Connection properties
	// (链接属性)
	property map[string]interface{}

	// Lock to protect the current property
	// (保护当前property的锁)
	propertyLock sync.Mutex

	// Which Connection Manager the current connection belongs to
	// (当前链接是属于哪个Connection Manager的)
	connManager giface.IConnManager

	// Hook function when the current connection is created
	// (当前连接创建时Hook函数)
	onConnStart func(conn giface.IConnection)

	// Hook function when the current connection is disconnected
	// (当前连接断开时的Hook函数)
	onConnStop func(conn giface.IConnection)

	// Data packet packaging method
	// (数据报文封包方式)
	packet giface.IDataPack

	// Last activity time
	// (最后一次活动时间)
	lastActivityTime time.Time

	// Framedecoder for solving fragmentation and packet sticking problems
	// (断粘包解码器)
	frameDecoder giface.IFrameDecoder

	// Heartbeat checker
	// (心跳检测器)
	hc giface.IHeartbeatChecker

	// Connection name, default to be the same as the name of the Server/Client that created the connection
	// (链接名称，默认与创建链接的Server/Client的Name一致)
	name string

	// Local address of the current connection
	// (当前链接的本地地址)
	localAddr string

	// Remote address of the current connection
	// (当前链接的远程地址)
	remoteAddr string

	// Close callback
	closeCallback callbacks

	// Close callback mutex
	closeCallbackMutex sync.RWMutex
}

// (创建一个Server服务端特性的连接的方法)
func newServerConn(server giface.IServer, conn net.Conn, connID uint64) giface.IConnection {
	// Initialize Conn properties
	c := &Connection{
		conn:            conn,
		connID:          connID,
		connIdStr:       strconv.FormatUint(connID, 10),
		startWriterFlag: 0,
		msgBuffChan:     nil,
		property:        nil,
		name:            server.ServerName(),
		localAddr:       conn.LocalAddr().String(),
		remoteAddr:      conn.RemoteAddr().String(),
	}

	lengthField := server.GetLengthField()
	if lengthField != nil {
		c.frameDecoder = ginterceptor.NewFrameDecoder(*lengthField)
	}

	// Inherited properties from server (从server继承过来的属性)
	c.packet = server.GetPacket()
	c.onConnStart = server.GetOnConnStart()
	c.onConnStop = server.GetOnConnStop()
	c.msgHandler = server.GetMsgHandler()

	// Bind the current Connection with the Server's ConnManager
	// (将当前的Connection与Server的ConnManager绑定)
	c.connManager = server.GetConnMgr()

	// Add the newly created Conn to the connection manager
	// (将新创建的Conn添加到链接管理中)
	server.GetConnMgr().Add(c)

	return c
}

// (创建一个Client服务端特性的连接的方法)
func newClientConn(client giface.IClient, conn net.Conn) giface.IConnection {
	c := &Connection{
		conn:            conn,
		connID:          0,  // client ignore
		connIdStr:       "", // client ignore
		startWriterFlag: 0,
		msgBuffChan:     nil,
		property:        nil,
		name:            client.GetName(),
		localAddr:       conn.LocalAddr().String(),
		remoteAddr:      conn.RemoteAddr().String(),
	}

	lengthField := client.GetLengthField()
	if lengthField != nil {
		c.frameDecoder = ginterceptor.NewFrameDecoder(*lengthField)
	}

	// Inherited properties from server (从client继承过来的属性)
	c.packet = client.GetPacket()
	c.onConnStart = client.GetOnConnStart()
	c.onConnStop = client.GetOnConnStop()
	c.msgHandler = client.GetMsgHandler()

	return c
}

// (写消息Goroutine， 用户将数据发送给客户端)
func (c *Connection) StartWriter() {
	glog.Ins().InfoF("Writer Goroutine is running")
	defer glog.Ins().InfoF("%s [conn Writer exit!]", c.RemoteAddr().String())

	for {
		select {
		case data, ok := <-c.msgBuffChan:
			if ok {
				if err := c.Send(data); err != nil {
					glog.Ins().ErrorF("Send Buff Data error:, %s Conn Writer exit", err)
					break
				}

			} else {
				glog.Ins().ErrorF("msgBuffChan is Closed")
				break
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// (读消息Goroutine，用于从客户端中读取数据)
func (c *Connection) StartReader() {
	glog.Ins().InfoF("[Reader Goroutine is running]")
	defer glog.Ins().InfoF("%s [conn Reader exit!]", c.RemoteAddr().String())
	defer c.Stop()
	defer func() {
		if err := recover(); err != nil {
			glog.Ins().ErrorF("connID=%d, panic err=%v", c.GetConnID(), err)
		}
	}()

	//Reduce buffer allocation times to improve efficiency
	// add by ray 2023-02-03
	buffer := make([]byte, gconf.GlobalObject.IOReadBuffSize)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:

			// read data from the connection's IO into the memory buffer
			// (从conn的IO中读取数据到内存缓冲buffer中)
			n, err := c.conn.Read(buffer)
			if err != nil {
				glog.Ins().ErrorF("read msg head [read datalen=%d], error = %s", n, err)
				return
			}
			glog.Ins().DebugF("read buffer %s \n", hex.EncodeToString(buffer[0:n]))

			// If normal data is read from the peer, update the heartbeat detection Active state
			// (正常读取到对端数据，更新心跳检测Active状态)
			if n > 0 && c.hc != nil {
				c.updateActivity()
			}

			// Deal with the custom protocol fragmentation problem, added by uuxia 2023-03-21
			// (处理自定义协议断粘包问题)
			if c.frameDecoder != nil {
				// Decode the 0-n bytes of data read
				// (为读取到的0-n个字节的数据进行解码)
				bufArrays := c.frameDecoder.Decode(buffer[0:n])
				if bufArrays == nil {
					continue
				}
				for _, bytes := range bufArrays {
					// glog.Ins().DebugF("read buffer %s \n", hex.EncodeToString(bytes))
					msg := gpack.NewMessage(uint32(len(bytes)), bytes)
					// Get the current client's Request data
					// (得到当前客户端请求的Request数据)
					req := GetRequest(c, msg)
					c.msgHandler.Execute(req)
				}
			} else {
				msg := gpack.NewMessage(uint32(n), buffer[0:n])
				// Get the current client's Request data
				// (得到当前客户端请求的Request数据)
				req := GetRequest(c, msg)
				c.msgHandler.Execute(req)
			}
		}
	}
}

// (启动连接，让当前连接开始工作)
func (c *Connection) Start() {
	defer func() {
		if err := recover(); err != nil {
			glog.Ins().ErrorF("Connection Start() error: %v", err)
		}
	}()
	c.ctx, c.cancel = context.WithCancel(context.Background())

	// Execute the hook method for processing business logic when creating a connection
	// (按照用户传递进来的创建连接时需要处理的业务，执行钩子方法)
	c.callOnConnStart()

	// Start heartbeating detection
	if c.hc != nil {
		c.hc.Start()
		c.updateActivity()
	}

	// 占用workerid
	c.workerID = useWorker(c)

	// Start the Goroutine for reading data from the client
	// (开启用户从客户端读取数据流程的Goroutine)
	go c.StartReader()

	select {
	case <-c.ctx.Done():
		c.finalizer()

		// 归还workerid
		freeWorker(c)
		return
	}
}

// Stop stops the connection and ends the current connection state.
// (停止连接，结束当前连接状态)
func (c *Connection) Stop() {
	c.cancel()
}

func (c *Connection) GetConnection() net.Conn {
	return c.conn
}

func (c *Connection) GetWsConn() *websocket.Conn {
	return nil
}

// Deprecated: use GetConnection instead
func (c *Connection) GetTCPConnection() net.Conn {
	return c.conn
}

func (c *Connection) GetConnID() uint64 {
	return c.connID
}

func (c *Connection) GetConnIdStr() string {
	return c.connIdStr
}

func (c *Connection) GetWorkerID() uint32 {
	return c.workerID
}

func (c *Connection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Connection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Connection) Flush() error {
	if c.isClosed() == true {
		return errors.New("connection closed when flush data")
	}
	return c.bufWriter.Flush()
}

func (c *Connection) Send(data []byte) error {
	if c.isClosed() == true {
		return errors.New("connection closed when send msg")
	}

	_, err := c.conn.Write(data)
	if err != nil {
		glog.Ins().ErrorF("SendMsg err data = %+v, err = %+v", data, err)
		return err
	}

	return nil
}

func (c *Connection) SendToQueue(data []byte) error {

	if c.msgBuffChan == nil && c.setStartWriterFlag() {
		c.msgBuffChan = make(chan []byte, gconf.GlobalObject.MaxMsgChanLen)
		// Start a Goroutine to write data back to the client
		// This method only reads data from the MsgBuffChan without allocating memory or starting a Goroutine
		// (开启用于写回客户端数据流程的Goroutine
		// 此方法只读取MsgBuffChan中的数据没调用SendBuffMsg可以分配内存和启用协程)
		go c.StartWriter()
	}

	idleTimeout := time.NewTimer(5 * time.Millisecond)
	defer idleTimeout.Stop()

	if c.isClosed() == true {
		return errors.New("Connection closed when send buff msg")
	}

	if data == nil {
		glog.Ins().ErrorF("Pack data is nil")
		return errors.New("Pack data is nil")
	}

	// Send timeout
	select {
	case <-c.ctx.Done():
		// Close all channels associated with the connection
		close(c.msgBuffChan)
		return errors.New("connection closed when send buff msg")
	case <-idleTimeout.C:
		return errors.New("send buff msg timeout")
	case c.msgBuffChan <- data:
		return nil
	}
}

// SendMsg directly sends Message data to the remote TCP client.
// (直接将Message数据发送数据给远程的TCP客户端)
func (c *Connection) SendMsg(msgID uint32, data []byte) error {

	if c.isClosed() == true {
		return errors.New("connection closed when send msg")
	}
	// Pack data and send it
	msg, err := c.packet.Pack(gpack.NewMsgPackage(msgID, data))
	if err != nil {
		glog.Ins().ErrorF("Pack error msg ID = %d", msgID)
		return errors.New("Pack error msg ")
	}

	err = c.Send(msg)
	if err != nil {
		glog.Ins().ErrorF("SendMsg err msg ID = %d, data = %+v, err = %+v", msgID, string(msg), err)
		return err
	}

	return nil
}

func (c *Connection) SendBuffMsg(msgID uint32, data []byte) error {
	msg, err := c.packet.Pack(gpack.NewMsgPackage(msgID, data))
	if err != nil {
		glog.Ins().ErrorF("Pack error msg ID = %d", msgID)
		return errors.New("Pack error msg ")
	}
	return c.SendToQueue(msg)

}

func (c *Connection) SetProperty(key string, value interface{}) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()
	if c.property == nil {
		c.property = make(map[string]interface{})
	}

	c.property[key] = value
}

func (c *Connection) GetProperty(key string) (interface{}, error) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()

	if value, ok := c.property[key]; ok {
		return value, nil
	}

	return nil, errors.New("no property found")
}

func (c *Connection) RemoveProperty(key string) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()

	delete(c.property, key)
}

func (c *Connection) Context() context.Context {
	return c.ctx
}

func (c *Connection) finalizer() {
	// Call the callback function registered by the user when closing the connection if it exists
	// (如果用户注册了该链接的	关闭回调业务，那么在此刻应该显示调用)
	c.callOnConnStop()

	// Stop the heartbeat detector associated with the connection
	if c.hc != nil {
		c.hc.Stop()
	}

	// Close the socket connection
	_ = c.conn.Close()

	// Remove the connection from the connection manager
	if c.connManager != nil {
		c.connManager.Remove(c)
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				glog.Ins().ErrorF("Conn finalizer panic: %v", err)
			}
		}()

		c.InvokeCloseCallbacks()
	}()

	glog.Ins().InfoF("Conn Stop()...ConnID = %d", c.connID)
}

func (c *Connection) callOnConnStart() {
	if c.onConnStart != nil {
		glog.Ins().InfoF("Gray CallOnConnStart....")
		c.onConnStart(c)
	}
}

func (c *Connection) callOnConnStop() {
	if c.onConnStop != nil {
		glog.Ins().InfoF("Gray CallOnConnStop....")
		c.onConnStop(c)
	}
}

func (c *Connection) IsAlive() bool {
	if c.isClosed() {
		return false
	}
	// Check the last activity time of the connection. If it's beyond the heartbeat interval,
	// then the connection is considered dead.
	// (检查连接最后一次活动时间，如果超过心跳间隔，则认为连接已经死亡)
	return time.Now().Sub(c.lastActivityTime) < gconf.GlobalObject.HeartbeatMaxDuration()
}

func (c *Connection) updateActivity() {
	c.lastActivityTime = time.Now()
}

func (c *Connection) SetHeartBeat(checker giface.IHeartbeatChecker) {
	c.hc = checker
}

func (c *Connection) LocalAddrString() string {
	return c.localAddr
}

func (c *Connection) RemoteAddrString() string {
	return c.remoteAddr
}

func (c *Connection) GetName() string {
	return c.name
}

func (c *Connection) GetMsgHandler() giface.IMsgHandler {
	return c.msgHandler
}

func (c *Connection) isClosed() bool {
	return c.ctx == nil || c.ctx.Err() != nil
}

func (c *Connection) setStartWriterFlag() bool {
	return atomic.CompareAndSwapInt32(&c.startWriterFlag, 0, 1)
}

func (s *Connection) AddCloseCallback(handler, key interface{}, f func()) {
	if s.isClosed() {
		return
	}
	s.closeCallbackMutex.Lock()
	defer s.closeCallbackMutex.Unlock()
	s.closeCallback.Add(handler, key, f)
}

func (s *Connection) RemoveCloseCallback(handler, key interface{}) {
	if s.isClosed() {
		return
	}
	s.closeCallbackMutex.Lock()
	defer s.closeCallbackMutex.Unlock()
	s.closeCallback.Remove(handler, key)
}

func (s *Connection) InvokeCloseCallbacks() {
	s.closeCallbackMutex.RLock()
	defer s.closeCallbackMutex.RUnlock()
	s.closeCallback.Invoke()
}

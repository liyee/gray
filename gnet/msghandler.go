package gnet

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/liyee/gtcp/gconf"
	"github.com/liyee/gtcp/giface"
	"github.com/liyee/gtcp/glog"
)

const (
	WorkerIDWithoutWorkerPool int = 0
)

type MsgHandler struct {
	Apis map[uint32]giface.IRouter //存放每个MsgID 所对应的处理方法的map属性

	WorkerPoolSize uint32 //业务工作Worker池的数量

	freeWorkers  map[uint32]struct{} //空闲worker集合，用于gconf.WorkerModeBind
	freeWorkerMu sync.Mutex

	TaskQueue []chan giface.IRequest //Worker负责取任务的消息队列

	// (责任链构造器)
	builder      *chainBuilder
	RouterSlices *RouterSlices
}

func newMsgHandler() *MsgHandler {
	var freeWorkers map[uint32]struct{}
	if gconf.GlobalObject.WorkerMode == gconf.WorkerModeBind {
		// Assign a workder to each link, avoid interactions when multiple links are processed by the same worker
		// MaxWorkerTaskLen can also be reduced, for example, 50
		// 为每个链接分配一个workder，避免同一worker处理多个链接时的互相影响
		// 同时可以减小MaxWorkerTaskLen，比如50，因为每个worker的负担减轻了
		gconf.GlobalObject.WorkerPoolSize = uint32(gconf.GlobalObject.MaxConn)
		freeWorkers = make(map[uint32]struct{}, gconf.GlobalObject.WorkerPoolSize)
		for i := uint32(0); i < gconf.GlobalObject.WorkerPoolSize; i++ {
			freeWorkers[i] = struct{}{}
		}
	}

	handler := &MsgHandler{
		Apis:           make(map[uint32]giface.IRouter),
		RouterSlices:   NewRouterSlices(),
		WorkerPoolSize: gconf.GlobalObject.WorkerPoolSize,
		// One worker corresponds to one queue (一个worker对应一个queue)
		TaskQueue:   make([]chan giface.IRequest, gconf.GlobalObject.WorkerPoolSize),
		freeWorkers: freeWorkers,
		builder:     newChainBuilder(),
	}

	// It is necessary to add the MsgHandle to the responsibility chain here, and it is the last link in the responsibility chain. After decoding in the MsgHandle, data distribution is done by router
	// (此处必须把 msghandler 添加到责任链中，并且是责任链最后一环，在msghandler中进行解码后由router做数据分发)
	handler.builder.Tail(handler)
	return handler
}
func useWorker(conn giface.IConnection) uint32 {
	var workerId uint32

	mh, _ := conn.GetMsgHandler().(*MsgHandler)
	if mh == nil {
		glog.Ins().ErrorF("useWorker failed, mh is nil")
		return 0
	}

	if gconf.GlobalObject.WorkerMode == gconf.WorkerModeBind {
		mh.freeWorkerMu.Lock()
		defer mh.freeWorkerMu.Unlock()

		for k := range mh.freeWorkers {
			delete(mh.freeWorkers, k)
			return k
		}
	} //(兼容client没有worker情况，解决除0的情况)
	if mh.WorkerPoolSize == 0 {
		workerId = 0
	} else {
		// Assign the worker responsible for processing the current connection based on the ConnID
		// Using a round-robin average allocation rule to get the workerID that needs to process this connection
		// (根据ConnID来分配当前的连接应该由哪个worker负责处理
		// 轮询的平均分配法则
		// 得到需要处理此条连接的workerID)
		workerId = uint32(conn.GetConnID() % uint64(mh.WorkerPoolSize))
	}

	return workerId
}
func freeWorker(conn giface.IConnection) {
	mh, _ := conn.GetMsgHandler().(*MsgHandler)
	if mh == nil {
		glog.Ins().ErrorF("useWorker failed, mh is nil")
		return
	}

	if gconf.GlobalObject.WorkerMode == gconf.WorkerModeBind {
		mh.freeWorkerMu.Lock()
		defer mh.freeWorkerMu.Unlock()

		mh.freeWorkers[conn.GetWorkerID()] = struct{}{}
	}
}
func (mh *MsgHandler) Intercept(chain giface.IChain) giface.IcResp {
	request := chain.Request()
	if request != nil {
		switch request.(type) {
		case giface.IRequest:
			iRequest := request.(giface.IRequest)
			if gconf.GlobalObject.WorkerPoolSize > 0 {
				// If the worker pool mechanism has been started, hand over the message to the worker for processing
				// (已经启动工作池机制，将消息交给Worker处理)
				mh.SendMsgToTaskQueue(iRequest)
			} else {

				// Execute the corresponding Handle method from the bound message and its corresponding processing method
				// (从绑定好的消息和对应的处理方法中执行对应的Handle方法)
				if !gconf.GlobalObject.RouterSlicesMode {
					go mh.doMsgHandler(iRequest, WorkerIDWithoutWorkerPool)
				} else if gconf.GlobalObject.RouterSlicesMode {
					go mh.doMsgHandlerSlices(iRequest, WorkerIDWithoutWorkerPool)
				}

			}
		}
	}

	return chain.Proceed(chain.Request())
}
func (mh *MsgHandler) SetHeadInterceptor(interceptor giface.IInterceptor) {
	if mh.builder != nil {
		mh.builder.Head(interceptor)
	}
}

func (mh *MsgHandler) AddInterceptor(interceptor giface.IInterceptor) {
	if mh.builder != nil {
		mh.builder.AddInterceptor(interceptor)
	}
}

// SendMsgToTaskQueue sends the message to the TaskQueue for processing by the worker
// (将消息交给TaskQueue,由worker进行处理)
func (mh *MsgHandler) SendMsgToTaskQueue(request giface.IRequest) {
	workerID := request.GetConnection().GetWorkerID()
	// glog.Ins().DebugF("Add ConnID=%d request msgID=%d to workerID=%d", request.GetConnection().GetConnID(), request.GetMsgID(), workerID)
	// Send the request message to the task queue
	mh.TaskQueue[workerID] <- request
	glog.Ins().DebugF("SendMsgToTaskQueue-->%s", hex.EncodeToString(request.GetData()))
}
func (mh *MsgHandler) doFuncHandler(request giface.IFuncRequest, workerID int) {
	defer func() {
		if err := recover(); err != nil {
			glog.Ins().ErrorF("workerID: %d doFuncRequest panic: %v", workerID, err)
		}
	}()
	// Execute the functional request (执行函数式请求)
	request.CallFunc()
}
func (mh *MsgHandler) doMsgHandler(request giface.IRequest, workerID int) {
	defer func() {
		if err := recover(); err != nil {
			glog.Ins().ErrorF("workerID: %d doMsgHandler panic: %v", workerID, err)
		}
	}()

	msgId := request.GetMsgID()
	handler, ok := mh.Apis[msgId]

	if !ok {
		glog.Ins().ErrorF("api msgID = %d is not FOUND!", request.GetMsgID())
		return
	}

	// Bind the Request request to the corresponding Router relationship
	// (Request请求绑定Router对应关系)
	request.BindRouter(handler)

	// Execute the corresponding processing method
	request.Call()

	// 执行完成后回收 Request 对象回对象池
	PutRequest(request)
}
func (mh *MsgHandler) Execute(request giface.IRequest) {
	// Pass the message to the responsibility chain to handle it through interceptors layer by layer and pass it on layer by layer.
	// (将消息丢到责任链，通过责任链里拦截器层层处理层层传递)
	mh.builder.Execute(request)
}

// AddRouter adds specific processing logic for messages
// (为消息添加具体的处理逻辑)
func (mh *MsgHandler) AddRouter(msgID uint32, router giface.IRouter) {
	// 1. Check whether the current API processing method bound to the msgID already exists
	// (判断当前msg绑定的API处理方法是否已经存在)
	if _, ok := mh.Apis[msgID]; ok {
		msgErr := fmt.Sprintf("repeated api , msgID = %+v\n", msgID)
		panic(msgErr)
	}
	// 2. Add the binding relationship between msg and API
	// (添加msg与api的绑定关系)
	mh.Apis[msgID] = router
	glog.Ins().InfoF("Add Router msgID = %d", msgID)
}
func (mh *MsgHandler) AddRouterSlices(msgId uint32, handler ...giface.RouterHandler) giface.IRouterSlices {
	mh.RouterSlices.AddHandler(msgId, handler...)
	return mh.RouterSlices
}

// Group routes into a group (路由分组)
func (mh *MsgHandler) Group(start, end uint32, Handlers ...giface.RouterHandler) giface.IGroupRouterSlices {
	return NewGroup(start, end, mh.RouterSlices, Handlers...)
}
func (mh *MsgHandler) Use(Handlers ...giface.RouterHandler) giface.IRouterSlices {
	mh.RouterSlices.Use(Handlers...)
	return mh.RouterSlices
}

func (mh *MsgHandler) doMsgHandlerSlices(request giface.IRequest, workerID int) {
	defer func() {
		if err := recover(); err != nil {
			glog.Ins().ErrorF("workerID: %d doMsgHandler panic: %v", workerID, err)
		}
	}()

	msgId := request.GetMsgID()
	handlers, ok := mh.RouterSlices.GetHandlers(msgId)
	if !ok {
		glog.Ins().ErrorF("api msgID = %d is not FOUND!", request.GetMsgID())
		return
	}

	request.BindRouterSlices(handlers)
	request.RouterSlicesNext()
	// 执行完成后回收 Request 对象回对象池
	PutRequest(request)
}

func (mh *MsgHandler) StartOneWorker(workerID int, taskQueue chan giface.IRequest) {
	glog.Ins().DebugF("Worker ID = %d is started.", workerID)
	// Continuously wait for messages in the queue
	// (不断地等待队列中的消息)
	for {
		select {
		// If there is a message, take out the Request from the queue and execute the bound business method
		// (有消息则取出队列的Request，并执行绑定的业务方法)
		case request := <-taskQueue:

			switch req := request.(type) {

			case giface.IFuncRequest:
				// Internal function call request (内部函数调用request)

				mh.doFuncHandler(req, workerID)

			case giface.IRequest: // Client message request

				if !gconf.GlobalObject.RouterSlicesMode {
					mh.doMsgHandler(req, workerID)
				} else if gconf.GlobalObject.RouterSlicesMode {
					mh.doMsgHandlerSlices(req, workerID)
				}
			}
		}
	}
}

func (mh *MsgHandler) StartWorkerPool() {
	// Iterate through the required number of workers and start them one by one
	// (遍历需要启动worker的数量，依此启动)
	for i := 0; i < int(mh.WorkerPoolSize); i++ {
		// A worker is started
		// Allocate space for the corresponding task queue for the current worker
		// (给当前worker对应的任务队列开辟空间)
		mh.TaskQueue[i] = make(chan giface.IRequest, gconf.GlobalObject.MaxWorkerTaskLen)

		// Start the current worker, blocking and waiting for messages to be passed in the corresponding task queue
		// (启动当前Worker，阻塞的等待对应的任务队列是否有消息传递进来)
		go mh.StartOneWorker(i, mh.TaskQueue[i])
	}
}

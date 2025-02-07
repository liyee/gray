package gnet

import (
	"errors"
	"strconv"

	"github.com/liyee/gtcp/giface"
	"github.com/liyee/gtcp/glog"
	"github.com/liyee/gtcp/gutils"
)

type ConnManager struct {
	connections gutils.ShardLockMaps
}

func newConnManager() *ConnManager {
	return &ConnManager{
		connections: gutils.NewShardLockMaps(),
	}
}

func (connMgr *ConnManager) Add(conn giface.IConnection) {

	connMgr.connections.Set(conn.GetConnIdStr(), conn) // 将conn连接添加到ConnManager中

	glog.Ins().DebugF("connection add to ConnManager successfully: conn num = %d", connMgr.Len())
}

func (connMgr *ConnManager) Remove(conn giface.IConnection) {

	connMgr.connections.Remove(conn.GetConnIdStr()) // 删除连接信息

	glog.Ins().DebugF("connection Remove ConnID=%d successfully: conn num = %d", conn.GetConnID(), connMgr.Len())
}

func (connMgr *ConnManager) Get(connID uint64) (giface.IConnection, error) {

	strConnId := strconv.FormatUint(connID, 10)
	if conn, ok := connMgr.connections.Get(strConnId); ok {
		return conn.(giface.IConnection), nil
	}

	return nil, errors.New("connection not found")
}

// Get2 It is recommended to use this method to obtain connection instances
func (connMgr *ConnManager) Get2(strConnId string) (giface.IConnection, error) {

	if conn, ok := connMgr.connections.Get(strConnId); ok {
		return conn.(giface.IConnection), nil
	}

	return nil, errors.New("connection not found")
}

func (connMgr *ConnManager) Len() int {

	length := connMgr.connections.Count()

	return length
}

func (connMgr *ConnManager) ClearConn() {

	// Stop and delete all connection information
	for item := range connMgr.connections.IterBuffered() {
		val := item.Val
		if conn, ok := val.(giface.IConnection); ok {
			// stop will eventually trigger the deletion of the connection,
			// no additional deletion is required
			conn.Stop()
		}
	}

	glog.Ins().InfoF("Clear All Connections successfully: conn num = %d", connMgr.Len())
}

func (connMgr *ConnManager) GetAllConnID() []uint64 {

	strConnIdList := connMgr.connections.Keys()
	ids := make([]uint64, 0, len(strConnIdList))

	for _, strId := range strConnIdList {
		connId, err := strconv.ParseUint(strId, 10, 64)
		if err == nil {
			ids = append(ids, connId)
		} else {
			glog.Ins().InfoF("GetAllConnID Id: %d, error: %v", connId, err)
		}
	}

	return ids
}

func (connMgr *ConnManager) GetAllConnIDStr() []string {
	return connMgr.connections.Keys()
}

func (connMgr *ConnManager) Range(cb func(uint64, giface.IConnection, interface{}) error, args interface{}) (err error) {

	connMgr.connections.IterCb(func(key string, v interface{}) {
		conn, _ := v.(giface.IConnection)
		connId, _ := strconv.ParseUint(key, 10, 64)
		err = cb(connId, conn, args)
		if err != nil {
			glog.Ins().InfoF("Range key: %v, v: %v, error: %v", key, v, err)
		}
	})

	return err
}

// Range2 It is recommended to use this method to 'Range'
func (connMgr *ConnManager) Range2(cb func(string, giface.IConnection, interface{}) error, args interface{}) (err error) {

	connMgr.connections.IterCb(func(key string, v interface{}) {
		conn, _ := v.(giface.IConnection)
		err = cb(conn.GetConnIdStr(), conn, args)
		if err != nil {
			glog.Ins().InfoF("Range2 key: %v, v: %v, error: %v", key, v, err)
		}
	})

	return err
}

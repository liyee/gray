package giface

type IConnManager interface {
	Add(IConnection)
	Remove(IConnection)
	Get(uint64) (IConnection, error)
	Get2(string) (IConnection, error)
	Len() int
	ClearConn()
	GetAllConnID() []uint64
	GetAllConnIDStr() []string
	Range(func(uint64, IConnection, interface{}) error, interface{}) error
	Range2(func(string, IConnection, interface{}) error, interface{}) error
}

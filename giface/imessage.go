package giface

type IMessage interface {
	GetDataLen() uint32
	GetMsgID() uint32
	GetData() []byte
	GetRawData() []byte

	SetMsgID(uint32)
	SetData([]byte)
	SetDataLen(uint32)
}

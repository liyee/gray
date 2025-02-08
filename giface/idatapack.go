package giface

type IDataPack interface {
	GetHeadLen() uint32                // Get the length of the message header(获取包头长度方法)
	Pack(msg IMessage) ([]byte, error) // Package message (封包方法)
	Unpack([]byte) (IMessage, error)   // Unpackage message(拆包方法)
}

const (
	// Gtcp standard packing and unpacking method (Gtcp 标准封包和拆包方式)
	GtcpDataPack    string = "gtcp_pack_tlv_big_endian"
	GtcpDataPackOld string = "gtcp_pack_ltv_little_endian"

	//...(+)
	//// Custom packing method can be added here(自定义封包方式在此添加)
)

const (
	// Gtcp default standard message protocol format(Gtcp 默认标准报文协议格式)
	GtcpMessage string = "gtcp_message"
)

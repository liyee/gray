package giface

type IDataPack interface {
	GetHeadLen() uint32                // Get the length of the message header(获取包头长度方法)
	Pack(msg IMessage) ([]byte, error) // Package message (封包方法)
	Unpack([]byte) (IMessage, error)   // Unpackage message(拆包方法)
}

const (
	// Gray standard packing and unpacking method (Gray 标准封包和拆包方式)
	GrayDataPack    string = "gray_pack_tlv_big_endian"
	GrayDataPackOld string = "gray_pack_ltv_little_endian"

	//...(+)
	//// Custom packing method can be added here(自定义封包方式在此添加)
)

const (
	// Gray default standard message protocol format(Gray 默认标准报文协议格式)
	GrayMessage string = "gray_message"
)

package gpack

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/liyee/gray/gconf"
	"github.com/liyee/gray/giface"
)

type DataPackLtv struct{}

// NewDataPackLtv initializes a packing and unpacking instance
// (封包拆包实例初始化方法)
func NewDataPackLtv() giface.IDataPack {
	return &DataPackLtv{}
}

// GetHeadLen returns the length of the message header
// (获取包头长度方法)
func (dp *DataPackLtv) GetHeadLen() uint32 {
	//ID uint32(4 bytes) +  DataLen uint32(4 bytes)
	return defaultHeaderLen
}

// Pack packs the message (compresses the data)
// (封包方法,压缩数据)
func (dp *DataPackLtv) Pack(msg giface.IMessage) ([]byte, error) {
	// Create a buffer to store the bytes
	// (创建一个存放bytes字节的缓冲)
	dataBuff := bytes.NewBuffer([]byte{})

	// Write the data length
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.GetDataLen()); err != nil {
		return nil, err
	}

	// Write the message ID
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.GetMsgID()); err != nil {
		return nil, err
	}

	// Write the data
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.GetData()); err != nil {
		return nil, err
	}

	return dataBuff.Bytes(), nil
}

// Unpack unpacks the message (decompresses the data)
// (拆包方法,解压数据)
func (dp *DataPackLtv) Unpack(binaryData []byte) (giface.IMessage, error) {
	// Create an ioReader for the input binary data
	dataBuff := bytes.NewReader(binaryData)

	// Only unpack the header information to obtain the data length and message ID
	// (只解压head的信息，得到dataLen和msgID)
	msg := &Message{}

	// Read the data length
	if err := binary.Read(dataBuff, binary.LittleEndian, &msg.DataLen); err != nil {
		return nil, err
	}

	// Read the message ID
	if err := binary.Read(dataBuff, binary.LittleEndian, &msg.ID); err != nil {
		return nil, err
	}

	// Check whether the data length exceeds the maximum allowed packet size
	// (判断dataLen的长度是否超出我们允许的最大包长度)
	if gconf.GlobalObject.MaxPacketSize > 0 && msg.GetDataLen() > gconf.GlobalObject.MaxPacketSize {
		return nil, errors.New("too large msg data received")
	}

	// Only the header data needs to be unpacked, and then another data read is performed from the connection based on the header length
	// (这里只需要把head的数据拆包出来就可以了，然后再通过head的长度，再从conn读取一次数据)
	return msg, nil
}

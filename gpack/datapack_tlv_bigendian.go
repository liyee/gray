package gpack

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/liyee/gray/gconf"
	"github.com/liyee/gray/giface"
)

var defaultHeaderLen uint32 = 8

type DataPack struct{}

// (封包拆包实例初始化方法)
func NewDataPack() giface.IDataPack {
	return &DataPack{}
}

// (获取包头长度方法)
func (dp *DataPack) GetHeadLen() uint32 {
	return defaultHeaderLen
}

// (封包方法,压缩数据)
func (dp *DataPack) Pack(msg giface.IMessage) ([]byte, error) {
	// Create a buffer to store the bytes
	// (创建一个存放bytes字节的缓冲)
	dataBuff := bytes.NewBuffer([]byte{})

	// Write the message ID
	if err := binary.Write(dataBuff, binary.BigEndian, msg.GetMsgID()); err != nil {
		return nil, err
	}

	// Write the data length
	if err := binary.Write(dataBuff, binary.BigEndian, msg.GetDataLen()); err != nil {
		return nil, err
	}

	// Write the data
	if err := binary.Write(dataBuff, binary.BigEndian, msg.GetData()); err != nil {
		return nil, err
	}

	return dataBuff.Bytes(), nil
}

// Unpack unpacks the message (decompresses the data)
// (拆包方法,解压数据)
func (dp *DataPack) Unpack(binaryData []byte) (giface.IMessage, error) {
	// Create an ioReader for the input binary data
	dataBuff := bytes.NewReader(binaryData)

	// Only unpack the header information to obtain the data length and message ID
	// (只解压head的信息，得到dataLen和msgID)
	msg := &Message{}

	// Read the data length
	if err := binary.Read(dataBuff, binary.BigEndian, &msg.ID); err != nil {
		return nil, err
	}

	// Read the message ID
	if err := binary.Read(dataBuff, binary.BigEndian, &msg.DataLen); err != nil {
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

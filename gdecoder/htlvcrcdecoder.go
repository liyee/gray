package gdecoder

import (
	"encoding/hex"
	"math"

	"github.com/liyee/gtcp/giface"
	"github.com/liyee/gtcp/glog"
)

const HEADER_SIZE = 5

type HtlvCrcDecoder struct {
	Head    byte   //HeaderCode(头码)
	Funcode byte   //FunctionCode(功能码)
	Length  byte   //DataLength(数据长度)
	Body    []byte //BodyData(数据内容)
	Crc     []byte //CRC校验
	Data    []byte //// Original data content(原始数据内容)
}

func NewHTLVCRCDecoder() giface.IDecoder {
	return &HtlvCrcDecoder{}
}

func (hcd *HtlvCrcDecoder) GetLengthField() *giface.LengthField {
	//+------+-------+---------+--------+--------+
	//| 头码  | 功能码 | 数据长度 | 数据内容 | CRC校验 |
	//| 1字节 | 1字节  | 1字节   | N字节   |  2字节  |
	//+------+-------+---------+--------+--------+
	//头码   功能码 数据长度      Body                         CRC
	//A2      10     0E        0102030405060708091011121314 050B
	//说明：
	//   1.数据长度len是14(0E),这里的len仅仅指Body长度;
	//
	//   lengthFieldOffset   = 2   (len的索引下标是2，下标从0开始) 长度字段的偏差
	//   lengthFieldLength   = 1   (len是1个byte) 长度字段占的字节数
	//   lengthAdjustment    = 2   (len只表示Body长度，程序只会读取len个字节就结束，但是CRC还有2byte没读呢，所以为2)
	//   initialBytesToStrip = 0   (这个0表示完整的协议内容，如果不想要A2，那么这里就是1) 从解码帧中第一次去除的字节数
	//   maxFrameLength      = 255 + 4(起始码、功能码、CRC) (len是1个byte，所以最大长度是无符号1个byte的最大值)
	return &giface.LengthField{
		MaxFrameLength:      math.MaxUint8 + 4,
		LengthFieldOffset:   2,
		LengthFieldLength:   1,
		LengthAdjustment:    2,
		InitialBytesToStrip: 0,
	}
}

func (hcd *HtlvCrcDecoder) decode(data []byte) *HtlvCrcDecoder {
	datasize := len(data)

	htlvData := HtlvCrcDecoder{
		Data: data,
	}

	// Parse the header
	htlvData.Head = data[0]
	htlvData.Funcode = data[1]
	htlvData.Length = data[2]
	htlvData.Body = data[3 : datasize-2]
	htlvData.Crc = data[datasize-2 : datasize]

	// CRC
	if !CheckCRC(data[:datasize-2], htlvData.Crc) {
		glog.Ins().DebugF("crc check error %s %s\n", hex.EncodeToString(data), hex.EncodeToString(htlvData.Crc))
		return nil
	}

	//glog.Ins().DebugF("2htlvData %s \n", hex.EncodeToString(htlvData.data))
	//glog.Ins().DebugF("HTLVCRC-DecodeData size:%d data:%+v\n", unsafe.Sizeof(htlvData), htlvData)

	return &htlvData
}

func (hcd *HtlvCrcDecoder) Intercept(chain giface.IChain) giface.IcResp {
	//1. Get the IMessage of zinx
	iMessage := chain.GetIMessage()
	if iMessage == nil {
		// Go to the next layer in the chain of responsibility
		return chain.ProceedWithIMessage(iMessage, nil)
	}

	//2. Get Data
	data := iMessage.GetData()
	//glog.Ins().DebugF("HTLVCRC-RawData size:%d data:%s\n", len(data), hex.EncodeToString(data))

	//3. If the amount of data read is less than the length of the header, proceed to the next layer directly.
	// (读取的数据不超过包头，直接进入下一层)
	if len(data) < HEADER_SIZE {
		return chain.ProceedWithIMessage(iMessage, nil)
	}

	//4. HTLV+CRC Decode
	htlvData := hcd.decode(data)

	//5. Set the decoded data back to the IMessage, the Zinx Router needs MsgID for addressing
	// (将解码后的数据重新设置到IMessage中, Zinx的Router需要MsgID来寻址)
	iMessage.SetMsgID(uint32(htlvData.Funcode))

	//6. Pass the decoded data to the next layer.
	// (将解码后的数据进入下一层)
	return chain.ProceedWithIMessage(iMessage, *htlvData)
}

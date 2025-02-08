package ginterceptor

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"sync"

	"github.com/liyee/gtcp/giface"
)

type FrameDecoder struct {
	giface.LengthField // Basic properties inherited from ILengthField

	LengthFieldEndOffset   int   // Offset of the end position of the length field (LengthFieldOffset+LengthFieldLength) (长度字段结束位置的偏移量)
	failFast               bool  // Fast failure (快速失败)
	discardingTooLongFrame bool  // true indicates discard mode is enabled, false indicates normal working mode (true 表示开启丢弃模式，false 正常工作模式)
	tooLongFrameLength     int64 // When the length of a packet exceeds maxLength, discard mode is enabled, and this field records the length of the data to be discarded (当某个数据包的长度超过maxLength，则开启丢弃模式，此字段记录需要丢弃的数据长度)
	bytesToDiscard         int64 // Records how many bytes still need to be discarded (记录还剩余多少字节需要丢弃)
	in                     []byte
	lock                   sync.Mutex
}

func NewFrameDecoder(lf giface.LengthField) giface.IFrameDecoder {

	frameDecoder := new(FrameDecoder)

	if lf.Order == nil {
		frameDecoder.Order = binary.BigEndian
	} else {
		frameDecoder.Order = lf.Order
	}
	frameDecoder.MaxFrameLength = lf.MaxFrameLength
	frameDecoder.LengthFieldOffset = lf.LengthFieldOffset
	frameDecoder.LengthFieldLength = lf.LengthFieldLength
	frameDecoder.LengthAdjustment = lf.LengthAdjustment
	frameDecoder.InitialBytesToStrip = lf.InitialBytesToStrip

	//self
	frameDecoder.LengthFieldEndOffset = lf.LengthFieldOffset + lf.LengthFieldLength
	frameDecoder.in = make([]byte, 0)

	return frameDecoder
}

func NewFrameDecoderByParams(maxFrameLength uint64, lengthFieldOffset, lengthFieldLength, lengthAdjustment, initialBytesToStrip int) giface.IFrameDecoder {
	return NewFrameDecoder(giface.LengthField{
		MaxFrameLength:      maxFrameLength,
		LengthFieldOffset:   lengthFieldOffset,
		LengthFieldLength:   lengthFieldLength,
		LengthAdjustment:    lengthAdjustment,
		InitialBytesToStrip: initialBytesToStrip,
		Order:               binary.BigEndian,
	})
}

func (d *FrameDecoder) fail(frameLength int64) {}

func (d *FrameDecoder) discardingTooLongFrameFunc(buffer *bytes.Buffer) {
	// Save the number of bytes still to be discarded
	// (保存还需丢弃多少字节)
	bytesToDiscard := d.bytesToDiscard

	// Get the number of bytes that can be discarded now, there may be a half package situation
	// (获取当前可以丢弃的字节数，有可能出现半包)
	localBytesToDiscard := math.Min(float64(bytesToDiscard), float64(buffer.Len()))

	// Discard (丢弃)
	buffer.Next(int(localBytesToDiscard))

	// Update the number of bytes still to be discarded (更新还需丢弃的字节数)
	bytesToDiscard -= int64(localBytesToDiscard)

	d.bytesToDiscard = bytesToDiscard

	// Determine if fast failure is needed, go back to the logic above (是否需要快速失败，回到上面的逻辑)
	d.failIfNecessary(false)
}

func (d *FrameDecoder) getUnadjustedFrameLength(buf *bytes.Buffer, offset int, length int, order binary.ByteOrder) int64 {
	// Value of the length field (长度字段的值)
	var frameLength int64

	arr := buf.Bytes()
	arr = arr[offset : offset+length]

	buffer := bytes.NewBuffer(arr)

	switch length {
	case 1:
		//byte
		var value uint8
		binary.Read(buffer, order, &value)
		frameLength = int64(value)
	case 2:
		//short
		var value uint16
		binary.Read(buffer, order, &value)
		frameLength = int64(value)
	case 3:
		// int occupies 32 bits, here take out the last 24 bits and return as int type
		// (int占32位，这里取出后24位，返回int类型)
		if order == binary.LittleEndian {
			n := uint(arr[0]) | uint(arr[1])<<8 | uint(arr[2])<<16
			frameLength = int64(n)
		} else {
			n := uint(arr[2]) | uint(arr[1])<<8 | uint(arr[0])<<16
			frameLength = int64(n)
		}
	case 4:
		//int
		var value uint32
		binary.Read(buffer, order, &value)
		frameLength = int64(value)
	case 8:
		//long
		binary.Read(buffer, order, &frameLength)
	default:
		panic(fmt.Sprintf("unsupported LengthFieldLength: %d (expected: 1, 2, 3, 4, or 8)", d.LengthFieldLength))
	}
	return frameLength
}

func (d *FrameDecoder) failOnNegativeLengthField(in *bytes.Buffer, frameLength int64, lengthFieldEndOffset int) {
	in.Next(lengthFieldEndOffset)
	panic(fmt.Sprintf("negative pre-adjustment length field: %d", frameLength))
}

func (d *FrameDecoder) failIfNecessary(firstDetectionOfTooLongFrame bool) {
	if d.bytesToDiscard == 0 {
		// Indicates that the data to be discarded has been discarded (说明需要丢弃的数据已经丢弃完成)
		// Save the length of the discarded data packet (保存一下被丢弃的数据包长度)
		tooLongFrameLength := d.tooLongFrameLength
		d.tooLongFrameLength = 0

		// Turn off discard mode (关闭丢弃模式)
		d.discardingTooLongFrame = false

		// failFast: Default is true (failFast：默认true)
		// firstDetectionOfTooLongFrame: Passed in as true (firstDetectionOfTooLongFrame：传入true)
		if !d.failFast || firstDetectionOfTooLongFrame {
			// Fast failure (快速失败)
			d.fail(tooLongFrameLength)
		}
	} else {
		// Indicates that the discard has not been completed yet (说明还未丢弃完成)
		if d.failFast && firstDetectionOfTooLongFrame {
			// Fast failure (快速失败)
			d.fail(d.tooLongFrameLength)
		}
	}
}

func (d *FrameDecoder) exceededFrameLength(in *bytes.Buffer, frameLength int64) {
	// Packet length - readable bytes (两种情况)
	// 1. Total length of the data packet is 100, readable bytes is 50, indicating that there are still 50 bytes to be discarded but have not been received yet
	// (数据包总长度为100，可读的字节数为50，说明还剩余50个字节需要丢弃但还未接收到)
	// 2. Total length of the data packet is 100, readable bytes is 150, indicating that the buffer already contains the entire data packet
	// (数据包总长度为100，可读的字节数为150，说明缓冲区已经包含了整个数据包)
	discard := frameLength - int64(in.Len())

	// Record the maximum length of the data packet (记录一下最大的数据包的长度)
	d.tooLongFrameLength = frameLength

	if discard < 0 {
		// Indicates the second case, directly discard the current data packet (说明是第2种情况，直接丢弃当前数据包)
		in.Next(int(frameLength))
	} else {
		// Indicates the first case, some data is still pending reception (说明是第1种情况，还有部分数据未接收到)
		// Enable discard mode (开启丢弃模式)
		d.discardingTooLongFrame = true

		// Record how many bytes need to be discarded next time (记录下次还需丢弃多少字节)
		d.bytesToDiscard = discard

		// Discard all data in the buffer (丢弃缓冲区所有数据)
		in.Next(in.Len())
	}

	// Update the status and determine if there is an error. (更新状态，判断是否有误)
	d.failIfNecessary(true)
}

func (d *FrameDecoder) failOnFrameLengthLessThanInitialBytesToStrip(in *bytes.Buffer, frameLength int64, initialBytesToStrip int) {
	in.Next(int(frameLength))
	panic(fmt.Sprintf("Adjusted frame length (%d) is less  than InitialBytesToStrip: %d", frameLength, initialBytesToStrip))
}

func (d *FrameDecoder) decode(buf []byte) []byte {
	in := bytes.NewBuffer(buf)

	// Determine if it is in discard mode (判断是否为丢弃模式)
	if d.discardingTooLongFrame {
		d.discardingTooLongFrameFunc(in)
	}

	// Determine if the number of readable bytes in the buffer is less than the offset of the length field
	// (判断缓冲区中可读的字节数是否小于长度字段的偏移量)
	if in.Len() < d.LengthFieldEndOffset {
		// Indicates that the length field packets are incomplete, half package
		// (说明长度字段的包都还不完整，半包)
		return nil
	}

	// --> If execution reaches here, it means that the value of the length field can be parsed <--
	// (执行到这，说明可以解析出长度字段的值了)

	// Calculate the offset of the length field
	// (计算出长度字段的开始偏移量)
	actualLengthFieldOffset := d.LengthFieldOffset

	// Get the value of the length field, excluding the adjustment value of lengthAdjustment
	// (获取长度字段的值，不包括lengthAdjustment的调整值)
	frameLength := d.getUnadjustedFrameLength(in, actualLengthFieldOffset, d.LengthFieldLength, d.Order)

	// If the data frame length is less than 0, it means it is an error data packet
	// (如果数据帧长度小于0，说明是个错误的数据包)
	if frameLength < 0 {
		// It will skip the number of bytes of this data packet and throw an exception
		// (内部会跳过这个数据包的字节数，并抛异常)
		d.failOnNegativeLengthField(in, frameLength, d.LengthFieldEndOffset)
	}

	// Apply the formula: Number of bytes after the length field = value of the length field + lengthAdjustment (如果数据帧长度小于0，说明是个错误的数据包)
	// frameLength is the value of the length field, plus lengthAdjustment equals the number of bytes after the length field (lengthFieldEndOffset is lengthFieldOffset+lengthFieldLength)
	// So the frameLength calculated in the end is the length of the entire data packet (那说明最后计算出的frameLength就是整个数据包的长度)
	frameLength += int64(d.LengthAdjustment) + int64(d.LengthFieldEndOffset)

	// Discard mode is turned on here (丢弃模式就是在这开启的)
	// If the data packet length is greater than the maximum length (如果数据包长度大于最大长度)
	if uint64(frameLength) > d.MaxFrameLength {
		// It has exceeded the maximum length of a single data frame, and the exceeded part is processed
		// (已经超过单次数据帧最大长度，对超过的部分进行处理)
		d.exceededFrameLength(in, frameLength)
		return nil
	}

	// --> If execution reaches here, it means normal mode <--
	// (执行到这, 说明是正常模式)

	// Size of the data packet (数据包的大小)
	frameLengthInt := int(frameLength)
	// Determine if the number of readable bytes in the buffer is less than the size of the data packet (判断缓冲区可读字节数是否小于数据包的字节数)
	if in.Len() < frameLengthInt {
		// Half package, will parse again later (半包，等会再来解析)
		return nil
	}

	// --> If execution reaches here, it means that the buffer already contains the entire data packet <--
	// (执行到这, 说明缓冲区的数据已经包含了数据包)

	// Whether the number of bytes to be skipped is greater than the length of the data packet (跳过的字节数是否大于数据包长度)
	if d.InitialBytesToStrip > frameLengthInt {
		// Will throw an exception if the length of the data packet is less than the number of bytes to be skipped (如果数据包长度小于跳过的字节数，将抛出异常)
		d.failOnFrameLengthLessThanInitialBytesToStrip(in, frameLength, d.InitialBytesToStrip)
	}

	// Skip the initialBytesToStrip bytes (跳过initialBytesToStrip个字节)
	in.Next(d.InitialBytesToStrip)

	// Decode (解码)
	// Get the real data length after skipping (获取跳过后的真实数据长度)
	actualFrameLength := frameLengthInt - d.InitialBytesToStrip

	// Extract the real data (提取真实的数据)
	buff := make([]byte, actualFrameLength)
	_, _ = in.Read(buff)

	return buff
}

func (d *FrameDecoder) Decode(buff []byte) [][]byte {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.in = append(d.in, buff...)
	resp := make([][]byte, 0)

	for {
		arr := d.decode(d.in)

		if arr != nil {
			// Indicates that a complete packet has been parsed
			// (证明已经解析出一个完整包)
			resp = append(resp, arr)
			_size := len(arr) + d.InitialBytesToStrip
			if _size > 0 {
				d.in = d.in[_size:]
			}
		} else {
			return resp
		}
	}
}

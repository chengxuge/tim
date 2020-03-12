package tim

import (
	"bytes"
	"encoding/binary"
)

type OpCode byte

type WebFrame struct {
	IsFrameEndOf  bool
	OpCode        OpCode
	PayloadLength int
	PayloadData   []byte
}

type WebPacket struct {
	MaskingKey []byte
	tmpKey     []byte
	Coder
}

const (
	ContinueFrame = OpCode(0)
	TextFrame     = OpCode(1)
	BinaryFrame   = OpCode(2)
	CloseFrame    = OpCode(8)
	PingFrame     = OpCode(9)
	PongFrame     = OpCode(10)
)

func (f *WebPacket) Marshal(writer *bytes.Buffer, msg interface{}) {
	if v, ok := msg.(*WebFrame); ok {
		writeFrame(f.MaskingKey, writer, v)
	} else {
		var noBodyLength = 10            //非body内容长度
		writer.WriteString("0x12345678") //预先填充10字节的长度头
		if f.MaskingKey != nil {
			writer.Write(f.MaskingKey)
			noBodyLength += 4
		}
		if err := f.Encode(writer, msg); err == nil {
			var payloadLength = writer.Len() - noBodyLength //0x12345678 or+ maskKey
			if payloadLength < 126 {
				writer.Next(8)
			} else if payloadLength < 0xFFFF {
				writer.Next(6)
			}

			var buf = writer.Bytes()
			var head byte = 1 << 7
			head |= byte(BinaryFrame)
			buf[0] = head //写入一个字节头

			var pl byte = 0
			if f.MaskingKey != nil {
				pl = 1 << 7
			}
			if payloadLength < 126 {
				pl |= byte(payloadLength)
				buf[1] = pl //开始写入长度
			} else if payloadLength < 0xFFFF {
				pl |= 126
				buf[1] = pl
				binary.BigEndian.PutUint16(buf[2:], uint16(payloadLength))
			} else {
				pl |= 127
				buf[1] = pl
				binary.BigEndian.PutUint64(buf[2:], uint64(payloadLength))
			}
			if f.MaskingKey != nil {
				masking(f.MaskingKey, buf[len(buf)-payloadLength:])
			}
		} else {
			Warn("web packet marshal error: %v", err)
			writer.Reset() //Marshal不支持写入多个消息
		}
	}
}

func (f *WebPacket) Unmarshal(reader *bytes.Buffer, msg *interface{}) (bool, error) {
	var buf = reader.Bytes() //获取收到的数据
	if oldLength := len(buf); oldLength >= 2 {
		var headData = buf[0]
		var isEndOf = (headData >> 7) > 0
		var opCode = OpCode(headData & 0x0F)

		var payload = buf[1]
		var isMask = (payload >> 7) > 0
		var payLength = int(payload & 0x7F)

		if payLength == 126 {
			if len(buf) >= 4 {
				payLength = int(binary.BigEndian.Uint16(buf[2:]))
				buf = buf[4:] //next 4
			} else {
				return false, nil
			}
		} else if payLength == 127 {
			if len(buf) >= 10 {
				payLength = int(binary.BigEndian.Uint64(buf[2:]))
				buf = buf[10:] //next 10
			} else {
				return false, nil
			}
		} else {
			buf = buf[2:] //next 2
		}

		if isMask {
			if len(buf) >= 4 {
				if f.tmpKey == nil {
					f.tmpKey = make([]byte, 4)
				}
				copy(f.tmpKey, buf)
				buf = buf[4:] //next 4
			} else {
				return false, nil
			}
		}

		if payLength == 0 {
			*msg = &WebFrame{
				IsFrameEndOf:  isEndOf,
				OpCode:        opCode,
				PayloadLength: 0,
				PayloadData:   nil,
			}
			reader.Next(oldLength - len(buf))
			return true, nil
		} else if len(buf) >= payLength {
			if isMask {
				masking(f.tmpKey, buf[:payLength])
			}
			if opCode == BinaryFrame && f.Coder != nil {
				reader.Next(oldLength - len(buf)) //过滤头部

				var mm, err = f.Decode(reader, payLength-2)
				if err == nil {
					*msg = mm
				}
				return true, err
			} else if opCode == BinaryFrame {
				var payData = make([]byte, payLength)
				copy(payData, buf)
				*msg = payData
				reader.Next(oldLength - len(buf[payLength:]))
				return true, nil
			} else if opCode == TextFrame {
				var payData = make([]byte, payLength)
				copy(payData, buf)
				*msg = string(payData)
				reader.Next(oldLength - len(buf[payLength:]))
				return true, nil
			} else {
				var payData = make([]byte, payLength) //这段的webFrame不支持
				copy(payData, buf)
				*msg = &WebFrame{
					IsFrameEndOf:  isEndOf,
					OpCode:        opCode,
					PayloadLength: payLength,
					PayloadData:   payData,
				}
				reader.Next(oldLength - len(buf[payLength:]))
				return true, nil //包含Opcode close
			}
		}
	}
	return false, nil
}

func writeFrame(maskingKey []byte, writer *bytes.Buffer, frame *WebFrame) {
	var head byte = 0
	if frame.IsFrameEndOf {
		head = 1 << 7
	}
	head |= byte(frame.OpCode)
	writer.WriteByte(head)

	var pl byte = 0
	if maskingKey != nil {
		pl = 1 << 7
	}
	if frame.PayloadLength < 126 {
		pl |= byte(frame.PayloadLength)
		writer.WriteByte(pl)
	} else if frame.PayloadLength < 0xFFFF {
		pl |= 126
		writer.WriteByte(pl)
		_ = binary.Write(writer, binary.BigEndian, uint16(frame.PayloadLength))
	} else {
		pl |= 127
		writer.WriteByte(pl)
		_ = binary.Write(writer, binary.BigEndian, uint64(frame.PayloadLength))
	}
	if maskingKey == nil {
		writer.Write(frame.PayloadData)
	} else {
		writer.Write(maskingKey)
		writer.Write(frame.PayloadData)
		masking(maskingKey, writer.Bytes()[(writer.Len()-frame.PayloadLength):])
	}
}

func masking(maskingKey, data []byte) {
	for i := 0; i < len(data); i++ {
		data[i] ^= maskingKey[i%len(maskingKey)]
	}
}

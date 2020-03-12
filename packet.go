package tim

import (
	"bytes"
	"encoding/binary"
)

type Packet interface {
	Marshal(writer *bytes.Buffer, msg interface{})
	Unmarshal(reader *bytes.Buffer, msg *interface{}) (bool, error)
}

type SizePacket struct {
	Coder
}

func (f *SizePacket) Marshal(writer *bytes.Buffer, msg interface{}) {
	var oldLength = writer.Len()
	writer.WriteString("0x")                      //预先填充长度头的位置
	if err := f.Encode(writer, msg); err == nil { //写入type头和实例数据
		var newLength = writer.Len()
		var buf = writer.Bytes()
		binary.BigEndian.PutUint16(buf[oldLength:], uint16(newLength-oldLength))
	} else {
		Warn("size packet marshal error: %v", err)
		writer.Truncate(oldLength) //出错保留之前写入的
	}
}

func (f *SizePacket) Unmarshal(reader *bytes.Buffer, msg *interface{}) (bool, error) {
	var aliveCount = reader.Len()
	if aliveCount >= 2 {
		var size = binary.BigEndian.Uint16(reader.Bytes()) //读取长读头
		if aliveCount >= int(size) {
			reader.Next(2)                              //过滤掉长度头
			var mm, err = f.Decode(reader, int(size-4)) //读取type头
			if err == nil {
				*msg = mm //防止空的mm赋值给msg，因为反序列化可能出错
			}
			return true, err
		}
	}
	return false, nil
}

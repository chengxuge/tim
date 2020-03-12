package tim

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
)

type PbCoder struct{}

func (f *PbCoder) Decode(reader *bytes.Buffer, size int) (interface{}, error) {
	var id, msg = decode(reader)
	if msg != nil {
		var buf = reader.Bytes()
		reader.Next(size)
		return msg, proto.Unmarshal(buf[:size], msg.(proto.Message)) //读取实例数据
	}
	return msg, fmt.Errorf("message %v is no supported", id)
}

func (f *PbCoder) Encode(writer *bytes.Buffer, msg interface{}) error {
	var ok, typeStr = encode(writer, msg)
	if ok {
		var buf, err = proto.Marshal(msg.(proto.Message))
		if err == nil {
			writer.Write(buf) //写入实例数据
		}
		return err
	}
	return fmt.Errorf("message %s is no supported", typeStr)
}

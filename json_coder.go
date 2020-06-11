package tim

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type JsonCoder struct{}

func (f *JsonCoder) Decode(reader *bytes.Buffer, bodySize int) (interface{}, error) {
	var id, msg = decode(reader)
	if msg != nil {
		var buf = reader.Bytes()
		reader.Next(bodySize)
		return msg, json.Unmarshal(buf[:bodySize], msg) //读取实例数据
	}
	return nil, fmt.Errorf("message %v is no supported", id)
}

func (f *JsonCoder) Encode(writer *bytes.Buffer, msg interface{}) error {
	var ok, typeStr = encode(writer, msg)
	if ok {
		var buf, err = json.Marshal(msg)
		if err == nil {
			writer.Write(buf) //写入实例数据
		}
		return err
	}
	return fmt.Errorf("message %s is no supported", typeStr)
}

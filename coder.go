package tim

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
)

type Coder interface {
	Decode(reader *bytes.Buffer, size int) (interface{}, error)
	Encode(writer *bytes.Buffer, msg interface{}) error
}

type JsonCoder struct{}

var (
	idType = make(map[int]reflect.Type)
	typeId = make(map[string]int)
	iotaId = 0 //累加id
)

const Iota = 0 //使用累加ID

func Register(id int, msg interface{}) {
	var msgType = reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		Fatal("register message pointer required")
	}
	if id == Iota {
		iotaId++
		id = iotaId //使用累加id的值
	} else {
		iotaId = id //重新设置累加ID
	}
	if _, ok := idType[id]; ok {
		Fatal("message %v is already registered", id)
	}
	idType[id] = msgType
	typeId[msgType.String()] = id
}

func (f *JsonCoder) Decode(reader *bytes.Buffer, size int) (interface{}, error) {
	var id, msg = decode(reader)
	if msg != nil {
		var buf = reader.Bytes()
		reader.Next(size)
		return msg, json.Unmarshal(buf[:size], msg) //读取实例数据
	}
	return msg, fmt.Errorf("message %v is no supported", id)
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

func decode(reader *bytes.Buffer) (int, interface{}) {
	var id int16
	_ = binary.Read(reader, binary.BigEndian, &id)
	if msgType := idType[int(id)]; msgType != nil {
		return int(id), reflect.New(msgType.Elem()).Interface()
	} else {
		return int(id), nil
	}
}

func encode(writer *bytes.Buffer, msg interface{}) (bool, string) {
	var msgType = reflect.TypeOf(msg)
	var typeStr = msgType.String()
	var id, ok = typeId[typeStr]
	if ok {
		_ = binary.Write(writer, binary.BigEndian, int16(id))
	}
	return ok, typeStr
}

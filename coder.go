package tim

import (
	"bytes"
	"encoding/binary"
	"reflect"
)

type Coder interface {
	Decode(reader *bytes.Buffer, bodySize int) (interface{}, error)
	Encode(writer *bytes.Buffer, msg interface{}) error
}

var (
	idType       = make(map[int16]reflect.Type)
	typeId       = make(map[string]int16)
	iotaId int16 = 0 //累加id
)

const Iota = 0 //使用累加ID

func Register(id int16, msg interface{}) {
	if msg == nil {
		Fatal("msg is nil")
	}
	var msgType = reflect.TypeOf(msg)
	if msgType.Kind() != reflect.Ptr {
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

func decode(reader *bytes.Buffer) (int16, interface{}) {
	var id int16
	_ = binary.Read(reader, binary.BigEndian, &id)
	if msgType := idType[id]; msgType != nil {
		return id, reflect.New(msgType.Elem()).Interface()
	} else {
		return id, nil
	}
}

func encode(writer *bytes.Buffer, msg interface{}) (bool, string) {
	var msgType = reflect.TypeOf(msg)
	var typeStr = msgType.String()
	var id, ok = typeId[typeStr]
	if ok {
		_ = binary.Write(writer, binary.BigEndian, id)
	}
	return ok, typeStr
}

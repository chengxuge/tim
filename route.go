package tim

import (
	"reflect"
)

type msgRouteInfo struct {
	mod   Module
	route MsgRouteFunc
}

type MsgRouteFunc func(*Agent, interface{})
type RpcRouteFunc func(Module, interface{}) interface{}

var (
	msgMap = make(map[string]*msgRouteInfo, 128)
	rpcMap = make(map[string]RpcRouteFunc, 128)
)

func MsgRoute(msg interface{}, mod Module, route MsgRouteFunc) {
	if msg == nil {
		Fatal("msg type required")
	}
	var msgType = reflect.TypeOf(msg)
	msgMap[msgType.String()] = &msgRouteInfo{
		mod:   mod,
		route: route,
	}
}

func RpcRoute(msg interface{}, route RpcRouteFunc) {
	if msg == nil {
		Fatal("msg type required")
	}
	var msgType = reflect.TypeOf(msg)
	rpcMap[msgType.String()] = route
}

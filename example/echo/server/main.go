package main

import (
	"fmt"
	//"net/http"
	//_ "net/http/pprof"
	"porsche/tim"
	"porsche/tim/example/msg"
)

func main() {
	tim.Register(tim.Iota, &msg.Ping{})
	tim.Register(tim.Iota, &msg.Pong{})

	var mod = &tim.BaseModule{
		Name: "server",
		OnError: func(module tim.Module, err interface{}) {
			fmt.Println(tim.GetPanicStack())
			module.Start(100, module)
		},
	}

	var sizePacket = &tim.SizePacket{Coder: &tim.JsonCoder{}}
	tim.ListenTcp(sizePacket, func(a *tim.Agent) {
		fmt.Println(a.Conn.RemoteAddr().String() + "connected...")
	}, func(a *tim.Agent) {
		fmt.Println(a.Conn.RemoteAddr().String() + "disconnect...")
	})
	tim.MsgRoute(&msg.Ping{}, mod, func(a *tim.Agent, m interface{}) {
		a.Send(&msg.Pong{ResponseData: m.(*msg.Ping).RequestData})
		//var i, x = 0, 0
		//i = i / x
	})
	mod.Start(100, mod)

	//http.ListenAndServe(":8088",nil)

	var cmd string
	_, _ = fmt.Scanf("%s", &cmd)
}

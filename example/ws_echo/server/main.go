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
		OnStart: func(mod tim.Module, restart bool) {
			fmt.Println("start", restart)
		},
		OnError: func(mod tim.Module, err interface{}) {
			fmt.Println(tim.GetPanicStack())
			mod.Start(100, mod)
		},
		OnClosed: func(mod tim.Module) {
			fmt.Println("close")
		},
	}

	var wsPacket = &tim.WebPacket{Coder: &tim.JsonCoder{}}
	tim.ListenWs(wsPacket, func(a *tim.Agent) {
		fmt.Println(a.Conn.RemoteAddr().String() + "connected...")
	}, func(a *tim.Agent) {
		fmt.Println(a.Conn.RemoteAddr().String() + "shake...")
		fmt.Println(a.WsCfg)
	}, func(a *tim.Agent) {
		fmt.Println(a.Conn.RemoteAddr().String() + "disconnect...")
	})
	tim.MsgRoute(&msg.Ping{}, mod, func(a *tim.Agent, m interface{}) {
		a.Send(&msg.Pong{ResponseData: m.(*msg.Ping).RequestData})
		//var i, x = 0, 0
		//i = i / x
	})
	tim.MsgRoute("", mod, func(a *tim.Agent, m interface{}) {
		a.Send(tim.WsText([]byte(m.(string))))
	})
	mod.Start(100, mod)

	//http.ListenAndServe(":8090", nil)

	var cmd string
	_, _ = fmt.Scanf("%s", &cmd)

	mod.Close(true)

	_, _ = fmt.Scanf("%s", &cmd)
}

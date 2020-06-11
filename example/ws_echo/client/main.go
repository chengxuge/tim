package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"porsche/tim"
	"porsche/tim/example/msg"
	"time"
)

func main() {
	tim.Register(tim.Iota, &msg.Ping{})
	tim.Register(tim.Iota, &msg.Pong{})

	var mod = &tim.BaseModule{
		Name: "client",
		OnError: func(module tim.Module, err interface{}) {
		},
	}

	var conn, err = net.Dial("tcp", "127.0.0.1:8800")
	if err != nil {
		tim.Fatal(err.Error())
	}
	var wsPacket = &tim.WebPacket{Coder: &tim.JsonCoder{}, MaskingKey: tim.NewMasking()}
	var agent = tim.NewWs(nil, conn, wsPacket, &tim.WsConfig{},
		func(a *tim.Agent) {

		}, func(a *tim.Agent) {
			fmt.Println(a.Conn.RemoteAddr().String() + "disconnect...")
		})
	tim.MsgRoute(&msg.Pong{}, mod, func(a *tim.Agent, m interface{}) {
		fmt.Println(m)
	})
	mod.TickFunc(time.Millisecond*100, func() {
		agent.Send(&msg.Ping{RequestData: time.Now().UnixNano()})
	})
	//mod.AfterFunc(5000*time.Millisecond, func() {
	//	agent.Send(&msg.Ping{RequestData:8888888})
	//})
	mod.Start(100, mod)

	http.ListenAndServe(":8091", nil)

	var cmd string
	_, _ = fmt.Scanf("%s", &cmd)
}

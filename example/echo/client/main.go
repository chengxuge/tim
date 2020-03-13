package main

import (
	"fmt"
	"net"
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

	var conn, err = net.Dial("tcp", "127.0.0.1:8088")
	if err != nil {
		tim.Fatal(err.Error())
	}
	var sizePacket = &tim.SizePacket{Coder: &tim.JsonCoder{}}
	var agent = tim.NewAgent(nil, conn, sizePacket, func(a *tim.Agent) {
		fmt.Println(a.Conn.RemoteAddr().String() + "disconnect...")
	})
	tim.MsgRoute(&msg.Pong{}, mod, func(a *tim.Agent, m interface{}) {
		fmt.Println(m)
	})
	mod.TickFunc(time.Millisecond*1000, func() {
		agent.Send(&msg.Ping{RequestData: time.Now().UnixNano()})
	})
	mod.Start(100, mod)

	var cmd string
	_, _ = fmt.Scanf("%s", &cmd)
}

package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"porsche/tim"
)

func main() {
	var wsPacket = &tim.WebPacket{}
	tim.ListenWs(wsPacket, func(a *tim.Agent) {
		fmt.Println(a.Conn.RemoteAddr().String() + "connected...")
	}, func(a *tim.Agent) {
		fmt.Println(a.Conn.RemoteAddr().String() + "shake...")
	}, func(a *tim.Agent) {
		fmt.Println(a.Conn.RemoteAddr().String() + "disconnect...")
	})

	tim.MsgRoute([]byte{}, nil, func(agent *tim.Agent, i interface{}) {
		fmt.Println(i)
	})
	tim.MsgRoute("", nil, func(agent *tim.Agent, i interface{}) {
		agent.Send(tim.WsText([]byte(i.(string))))
		fmt.Println(i)
	})

	http.ListenAndServe(":8090", nil)

	var cmd string
	_, _ = fmt.Scanf("%s", &cmd)
}

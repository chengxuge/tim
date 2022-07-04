package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"porsche/tim"
)

func main() {
	tim.MsgRoute([]byte{}, nil, func(a *tim.Agent, i interface{}) {
		var bytes, _ = i.([]byte)
		a.Send(bytes)
	})
	tim.ListenTcp(nil, func(a *tim.Agent) {

	}, func(a *tim.Agent) {
		a.Close()
	})

	http.ListenAndServe(":8091", nil)

	var cmd string
	_, _ = fmt.Scanf("%s", &cmd)
}

package tim

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"
)

type Module interface {
	ExeRpc(msg interface{}, timeout time.Duration) (interface{}, error)
	ExeMsg(route MsgRouteFunc, agent *Agent, msg interface{}) bool
	TickFunc(d time.Duration, f func()) (close chan struct{})
	AfterFunc(d time.Duration, f func()) *time.Timer
	Start(chanSize int, mod Module)
	Exe(f func()) bool
	Close(force bool)
	IsClosed() bool
}

type modInfo struct {
	Call  MsgRouteFunc
	Msg   interface{}
	Agent *Agent
}

type rpcInfo struct {
	Call RpcRouteFunc
	Msg  interface{}
	Ctx  context.Context
	Ret  chan interface{}
}

type BaseModule struct {
	Name         string                            //模块名称
	OnStart      func(mod Module, restart bool)    //开始运行
	OnError      func(mod Module, err interface{}) //错误处理
	OnClosed     func(mod Module)                  //模块关闭
	msgChan      chan interface{}                  //消息队列
	sendMu       sync.Mutex                        //同步锁
	isClosed     bool                              //是否已关闭
	isForceClose bool                              //立刻关闭，不等待chan处理完
}

func (f *BaseModule) AfterFunc(d time.Duration, f1 func()) *time.Timer {
	return time.AfterFunc(d, func() {
		f.Exe(f1)
	})
}

func (f *BaseModule) TickFunc(d time.Duration, f1 func()) (close chan struct{}) {
	var ticker = time.NewTicker(d)
	close = make(chan struct{})
	go func(c <-chan time.Time) {
		for {
			select {
			case <-c:
				f.Exe(f1)
			case <-close:
				ticker.Stop()
				return
			}
		}
	}(ticker.C)
	return close
}

func (f *BaseModule) ExeRpc(msg interface{}, timeout time.Duration) (interface{}, error) {
	if msg == nil {
		return nil, errors.New("msg is nil")
	}
	f.sendMu.Lock()
	if !f.isClosed {
		var t = reflect.TypeOf(msg)
		if route, ok := rpcMap[t.String()]; ok {
			var ctx, _ = context.WithTimeout(context.Background(), timeout)
			var rpc = &rpcInfo{
				Call: route,
				Msg:  msg,
				Ctx:  ctx,
				Ret:  make(chan interface{}),
			}
			f.msgChan <- rpc
			f.sendMu.Unlock()

			select {
			case <-ctx.Done():
				close(rpc.Ret)
				return nil, ctx.Err()
			case ret := <-rpc.Ret:
				close(rpc.Ret)
				return ret, nil
			}
		} else {
			f.sendMu.Unlock()
			return nil, errors.New("msg not route")
		}
	} else {
		f.sendMu.Unlock()
		return nil, errors.New("module is closed")
	}
}

func (f *BaseModule) ExeMsg(route MsgRouteFunc, agent *Agent, msg interface{}) bool {
	var result = false
	f.sendMu.Lock()
	if !f.isClosed {
		f.msgChan <- &modInfo{
			Call:  route,
			Msg:   msg,
			Agent: agent,
		}
		result = true
	}
	f.sendMu.Unlock()
	return result
}

func (f *BaseModule) Exe(f1 func()) bool {
	var result = false
	f.sendMu.Lock()
	if !f.isClosed {
		f.msgChan <- f1
		result = true
	}
	f.sendMu.Unlock()
	return result
}

func (f *BaseModule) Start(chanSize int, mod Module) {
	f.sendMu.Lock()
	defer f.sendMu.Unlock()
	if !f.isClosed {
		var restart = false
		if f.msgChan == nil {
			f.msgChan = make(chan interface{}, chanSize)
		} else {
			restart = true //重新启动
		}

		go func(c <-chan interface{}, restart bool) {
			//错误恢复代码
			defer func() {
				if err := recover(); err != nil {
					Error("%s\n%s", err, GetPanicStack()) //未知错误，记录到log文件

					if f.OnError != nil {
						f.OnError(f, err) //这里可调用Start恢复
					}
				}
			}()

			if f.OnStart != nil {
				f.OnStart(f, restart)
			}

			//逻辑处理代码
			for msg := range c {
				if v, ok := msg.(*modInfo); ok {
					v.Call(v.Agent, v.Msg)
				} else if v, ok := msg.(*rpcInfo); ok {
					select {
					case <-v.Ctx.Done():
						continue
					default:
						v.Ret <- v.Call(mod, v.Msg)
					}
				} else if f1, ok := msg.(func()); ok {
					f1() //timer or ticker等执行的函数
				}
				if f.isClosed && f.isForceClose {
					break //立刻关闭，不处理后续消息
				}
			}

			if f.OnClosed != nil {
				f.OnClosed(f)
			}
		}(f.msgChan, restart)
	}
}

func (f *BaseModule) Close(force bool) {
	f.sendMu.Lock()
	defer f.sendMu.Unlock()
	if !f.isClosed {
		f.isClosed = true
		f.isForceClose = force

		close(f.msgChan)
	}
}

func (f *BaseModule) IsClosed() bool {
	f.sendMu.Lock()
	defer f.sendMu.Unlock()
	return f.isClosed
}

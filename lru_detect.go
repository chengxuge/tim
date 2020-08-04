package tim

import (
	"container/list"
	"sync"
	"time"
)

type lruDetect struct {
	tick     *time.Ticker
	nodeList *list.List
	mu       sync.Mutex
}

var pLruDetect *lruDetect

func startLRUDetect(timeOut, interval time.Duration) {
	if pLruDetect == nil {
		pLruDetect = &lruDetect{
			tick:     time.NewTicker(interval),
			nodeList: list.New(),
		}
		pLruDetect.run(timeOut)
	}
	Info("timeOut: %v interval: %v", timeOut, interval)
}

func (f *lruDetect) run(timeOut time.Duration) {
	go func(t time.Duration, c <-chan time.Time) {
		for curTime := range c {
			f.mu.Lock()
			for e := f.nodeList.Back(); e != nil; e = e.Prev() {
				var a = e.Value.(*Agent)
				if curTime.Sub(a.lastTime) > t {
					_ = a.Conn.Close()
				} else {
					break
				}
			}
			f.mu.Unlock()
		}
	}(timeOut, f.tick.C)
}

func (f *lruDetect) update(a *Agent, now time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var e = a.lruNode
	if e != nil {
		f.nodeList.MoveToFront(e)
		a.lastTime = now
	} else {
		e = f.nodeList.PushFront(a)
		a.lruNode = e
		a.lastTime = now
	}
}

func (f *lruDetect) delete(a *Agent) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var e = a.lruNode
	if e != nil {
		f.nodeList.Remove(e)
		a.lruNode = nil
	}
}

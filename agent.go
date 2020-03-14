package tim

import (
	"bytes"
	"container/list"
	"net"
	"reflect"
	"sync"
	"time"
	"unsafe"
)

type Agent struct {
	Conn     net.Conn               //连接
	Packet   Packet                 //消息解析协议
	Tags     map[string]interface{} //会话保存数据
	WsCfg    *WsConfig              //websocket配置
	sendMu   sync.Mutex             //安全锁
	sendChan chan interface{}       //发送通道
	lruNode  *list.Element          //LRU链表节点
	lastTime time.Time              //最后检查时间
	wdTime   time.Time              //最后窗口时间
	wdCount  int                    //窗口期消息数量
	isShake  bool                   //是否握手成功
	isClosed bool                   //是否已关闭
}

type WsConfig struct {
	Host     string //主机
	Path     string //路径
	Origin   string //来源
	Protocol string //协议
}

func NewAgent(tags map[string]interface{}, conn net.Conn, packet Packet, onClose func(*Agent)) *Agent {
	var agent = &Agent{
		Conn:   conn,
		Packet: packet,
	}
	if pLruDetect != nil {
		pLruDetect.update(agent, time.Now())
	}
	agent.run(nil, onClose, tags)
	return agent
}

func newWs(conn net.Conn, packet *WebPacket, onShake, onClose func(*Agent)) *Agent {
	if packet == nil {
		Fatal("packet required")
	}
	if packet.MaskingKey != nil {
		Fatal("masking key nil required")
	}
	var agent = &Agent{
		Conn:   conn,
		Packet: packet,
		WsCfg:  &WsConfig{},
	}
	if pLruDetect != nil {
		pLruDetect.update(agent, time.Now())
	}
	agent.run(onShake, onClose, nil)
	return agent
}

func NewWs(tags map[string]interface{}, conn net.Conn, packet *WebPacket, wsCfg *WsConfig, onShake, onClose func(*Agent)) *Agent {
	if packet == nil {
		Fatal("packet required")
	}
	if len(packet.MaskingKey) != 4 {
		Fatal("masking key four bytes required")
	}
	var agent = &Agent{
		Conn:   conn,
		Packet: packet,
		WsCfg:  wsCfg,
	}
	if pLruDetect != nil {
		pLruDetect.update(agent, time.Now())
	}
	agent.run(onShake, onClose, tags)
	return agent
}

func (f *Agent) run(onShake, onClose func(*Agent), tags map[string]interface{}) {
	if tags == nil {
		f.Tags = make(map[string]interface{})
	} else {
		f.Tags = tags
	}
	f.sendChan = make(chan interface{}, 128)
	go f.goReceive(onShake, onClose)
	go f.goSend()
}

func (f *Agent) validateWd(now time.Time) bool {
	f.wdCount++
	if f.wdCount <= svrCfg.WindowNum {
		return true //在窗口期内小于最大消息数量
	} else if now.Sub(f.wdTime).Seconds() >= float64(svrCfg.WindowSec) {
		f.wdTime = now //新窗口期
		f.wdCount = 1
		return true //超过窗口期
	} else {
		return false
	}
}

func (f *Agent) Send(msg interface{}) {
	f.sendMu.Lock()
	if !f.isClosed {
		f.sendChan <- msg
	}
	f.sendMu.Unlock()
}

func (f *Agent) ClearTags() {
	for k := range f.Tags {
		delete(f.Tags, k)
	}
}

func (f *Agent) Close() {
	f.sendMu.Lock()
	if !f.isClosed {
		f.isShake = false
		f.isClosed = true
		close(f.sendChan)
	}
	f.sendMu.Unlock()
}

func (f *Agent) goReceive(onShake, onClose func(*Agent)) {
	var buffSize = svrCfg.BuffSize //获取设置的缓冲大小
	var reader = bytes.NewBuffer(make([]byte, 0, buffSize))
	var msg interface{}
	for {
		reader.Grow(buffSize) //扩充容量，不能改变位置，握手需要buff
		var b = reader.Bytes()
		var l = len(b)
		b = b[l : l+buffSize]
		n, err := f.Conn.Read(b)
		if err == nil && n > 0 {
			var now = time.Now()
			if pLruDetect != nil {
				pLruDetect.update(f, now)
			}
			setLength(reader, l+n) //设置新的length
			if !f.isShake && f.WsCfg != nil {
				if f.Packet.(*WebPacket).MaskingKey == nil {
					var b, wsCfg, ok = serverWebSocket(reader)
					if ok {
						_, _ = f.Conn.Write([]byte(getResponseOK(b, wsCfg.Protocol)))
						*f.WsCfg = *wsCfg //设置wsConfig信息
						f.isShake = true
						if onShake != nil {
							onShake(f)
						}
					} else {
						continue
					}
				} else {
					if clientWebSocket(reader) {
						f.isShake = true
						if onShake != nil {
							onShake(f)
						}
					} else {
						continue
					}
				}
			}

			if f.Packet != nil {
				for i := 0; ; i++ {
					var ret, err = f.Packet.Unmarshal(reader, &msg)
					if ret {
						if err == nil {
							//Debug("receive info:%#v", msg)

							if svrCfg.WindowSec == 0 || f.validateWd(now) {
								var t = reflect.TypeOf(msg)
								if info, ok := msgMap[t.String()]; ok {
									if mod := info.mod; mod != nil {
										mod.ExeMsg(info.route, f, msg)
									} else {
										info.route(f, msg)
									}
								}
							} else {
								Warn("window abandon msg:%#v", msg)
							}
							msg = nil //reset pointer
						} else {
							Warn("unmarshal error: %v", err) //解析包错误
							reader.Reset()                   //清除所有数据
							f.Close()                        //踢掉连接
							break
						}
						if reader.Len() > 0 {
							continue //还有数据继续读
						} else {
							reader.Reset() //已读完清理
						}
					} else if reader.Len() > svrCfg.MaxReadBytes {
						Warn("to max read bytes: %v", svrCfg.MaxReadBytes) //超过允许长度
						reader.Reset()                                     //清除所有数据
						f.Close()                                          //踢掉连接
						break
					} else if i > 0 {
						clearBuffer(reader) //i>0 读出过包,需要清理,防被半包耗死内存
					}
					break //没有完整的包
				}
			} else {
				if info, ok := msgMap["[]uint8"]; ok {
					if mod := info.mod; mod != nil {
						mod.ExeMsg(info.route, f, reader.Bytes())
					} else {
						info.route(f, reader.Bytes())
					}
				}
				reader.Reset() //清除未能解析所有数据
			}
		} else {
			if pLruDetect != nil {
				pLruDetect.delete(f)
			}
			if onClose != nil {
				onClose(f)
			}
			break //连接断开
		}
	}
}

func (f *Agent) goSend() {
	if f.WsCfg != nil && f.Packet.(*WebPacket).MaskingKey != nil {
		//发送websocket客户端握手请求
		_, _ = f.Conn.Write([]byte(getRequest(f.WsCfg)))
	}

	var writer = bytes.NewBuffer(make([]byte, 0, svrCfg.BuffSize))
	for msg := range f.sendChan {
		//Debug("send info:%#v", msg)

		if f.Packet != nil {
			f.Packet.Marshal(writer, msg)
			_, _ = writer.WriteTo(f.Conn)
		} else {
			switch msg.(type) {
			case []byte:
				_, _ = f.Conn.Write(msg.([]byte))
			case string:
				_, _ = f.Conn.Write([]byte(msg.(string)))
			}
		}
	}
	_ = f.Conn.Close() //将待发送的消息发送完后，再关闭
}

type buffer struct {
	buf      []byte // contents are the bytes buf[off : len(buf)]
	off      int    // read at &buf[off], write at &buf[len(buf)]
	lastRead int8   // last read operation, so that Unread* can work correctly.
}

func setLength(reader *bytes.Buffer, length int) {
	var ptr = (*buffer)(unsafe.Pointer(reader))
	ptr.buf = ptr.buf[:ptr.off+length]
}

func clearBuffer(reader *bytes.Buffer) {
	var ptr = (*buffer)(unsafe.Pointer(reader))
	var buf = ptr.buf[ptr.off:]
	ptr.buf = ptr.buf[:len(buf)]
	copy(ptr.buf, buf)
	ptr.off = 0
}

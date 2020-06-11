package tim

import (
	"crypto/tls"
	"net"
	"strings"
	"sync"
)

var (
	tcpLn, tlsLn, wsLn, wssLn net.Listener
	wgCons                    sync.WaitGroup
	consLock                  sync.Mutex
	tcpCons                   = make(map[string]*net.TCPConn, 2048)
	tlsCons                   = make(map[string]*tls.Conn, 2048)
	ipNumMap                  = make(map[string]float64, 2048)
)

func checkConn(conn net.Conn, network string) bool {
	consLock.Lock()
	defer consLock.Unlock()
	if (len(tcpCons) + len(tlsCons)) >= svrCfg.MaxConnNum {
		Warn("%s超过允许连接数量,已断开: %s", network, conn.RemoteAddr())
		_ = conn.Close()
		return false
	}
	var addrStr = conn.RemoteAddr().String()
	var ipEnd = strings.LastIndex(addrStr, ":")
	var ip = addrStr[:ipEnd]
	var whiteNum = svrCfg.IpWhiteList[ip]
	if num := ipNumMap[ip]; int(num) >= svrCfg.MaxIpConnNum && num >= whiteNum {
		Warn("Ip超过允许连接数量,已断开: %s", conn.RemoteAddr())
		_ = conn.Close()
		return false
	}
	if network == "tcp" || network == "ws" {
		tcpCons[addrStr] = conn.(*net.TCPConn)
	} else {
		tlsCons[addrStr] = conn.(*tls.Conn)
	}
	ipNumMap[ip]++
	return true
}

func closeConn(conn net.Conn, network string) {
	consLock.Lock()
	defer consLock.Unlock()
	var addrStr = conn.RemoteAddr().String()
	if network == "tcp" || network == "ws" {
		delete(tcpCons, addrStr)
	} else {
		delete(tlsCons, addrStr)
	}
	var ip = addrStr[:strings.LastIndex(addrStr, ":")]
	ipNumMap[ip]--
	if num := ipNumMap[ip]; num == 0 {
		delete(ipNumMap, ip)
	}
}

func startTcp(packet Packet, onConn, onClose func(*Agent)) bool {
	if svrCfg.TcpAddr != "" {
		var err error
		tcpLn, err = net.Listen("tcp", svrCfg.TcpAddr)
		if err != nil {
			Fatal(err.Error())
		}
		go func() {
			for {
				var conn, err = tcpLn.Accept()
				if err != nil {
					break
				}
				if checkConn(conn, "tcp") {
					wgCons.Add(1)
				} else {
					continue
				}
				var a = NewAgent(nil, conn, packet, func(a *Agent) {
					closeConn(conn, "tcp")
					wgCons.Done()
					if onClose != nil {
						onClose(a)
					}
				})
				if onConn != nil {
					onConn(a)
				}
			}
		}()
		return true
	}
	return false
}

func startTls(packet Packet, onConn, onClose func(*Agent)) bool {
	if svrCfg.TlsAddr != "" {
		if svrCfg.CertFile == "" || svrCfg.KeyFile == "" {
			Fatal("tls files error")
		}
		var cert, err = tls.LoadX509KeyPair(svrCfg.CertFile, svrCfg.KeyFile)
		if err != nil {
			Fatal(err.Error())
		}
		var config = &tls.Config{Certificates: []tls.Certificate{cert}}
		tlsLn, err = tls.Listen("tcp", svrCfg.TlsAddr, config)
		if err != nil {
			Fatal(err.Error())
		}
		go func() {
			for {
				var conn, err = tlsLn.Accept()
				if err != nil {
					break
				}
				if checkConn(conn, "tls") {
					wgCons.Add(1)
				} else {
					continue
				}
				var a = NewAgent(nil, conn, packet, func(a *Agent) {
					closeConn(conn, "tls")
					wgCons.Done()
					if onClose != nil {
						onClose(a)
					}
				})
				if onConn != nil {
					onConn(a)
				}
			}
		}()
		return true
	}
	return false
}

func startWs(packet *WebPacket, onConn, onShake, onClose func(*Agent)) bool {
	if svrCfg.WsAddr != "" {
		var err error
		wsLn, err = net.Listen("tcp", svrCfg.WsAddr)
		if err != nil {
			Fatal(err.Error())
		}
		go func() {
			for {
				var conn, err = wsLn.Accept()
				if err != nil {
					break
				}
				if checkConn(conn, "ws") {
					wgCons.Add(1)
				} else {
					continue
				}
				var a = newWs(conn, packet, onShake, func(a *Agent) {
					closeConn(conn, "ws")
					wgCons.Done()
					if onClose != nil {
						onClose(a)
					}
				})
				if onConn != nil {
					onConn(a)
				}
			}
		}()
		return true
	}
	return false
}

func startWss(packet *WebPacket, onConn, onShake, onClose func(*Agent)) bool {
	if svrCfg.WssAddr != "" {
		if svrCfg.CertFile == "" || svrCfg.KeyFile == "" {
			Fatal("tls files error")
		}
		var cert, err = tls.LoadX509KeyPair(svrCfg.CertFile, svrCfg.KeyFile)
		if err != nil {
			Fatal(err.Error())
		}
		var config = &tls.Config{Certificates: []tls.Certificate{cert}}
		wssLn, err = tls.Listen("tcp", svrCfg.WssAddr, config)
		if err != nil {
			Fatal(err.Error())
		}
		go func() {
			for {
				var conn, err = wssLn.Accept()
				if err != nil {
					break
				}
				if checkConn(conn, "wss") {
					wgCons.Add(1)
				} else {
					continue
				}
				var a = newWs(conn, packet, onShake, func(a *Agent) {
					closeConn(conn, "wss")
					wgCons.Done()
					if onClose != nil {
						onClose(a)
					}
				})
				if onConn != nil {
					onConn(a)
				}
			}
		}()
		return true
	}
	return false
}

func ListenTcp(packet Packet, onConn, onClose func(*Agent)) {
	if startTcp(packet, onConn, onClose) {
		Info("tcp:%s 正在监听中", svrCfg.TcpAddr)
	}
	if startTls(packet, onConn, onClose) {
		Info("tls:%s 正在监听中", svrCfg.TlsAddr)
	}
}

func ListenWs(wsPacket *WebPacket, onConn, onShake, onClose func(*Agent)) {
	if startWs(wsPacket, onConn, onShake, onClose) {
		Info("ws:%s 正在监听中", svrCfg.WsAddr)
	}
	if startWss(wsPacket, onConn, onShake, onClose) {
		Info("wss:%s 正在监听中", svrCfg.WssAddr)
	}
}

func Shutdown() {
	Info("tim 正在停止")
	if tcpLn != nil {
		_ = tcpLn.Close()
	}
	if tlsLn != nil {
		_ = tlsLn.Close()
	}
	if wsLn != nil {
		_ = wsLn.Close()
	}
	if wssLn != nil {
		_ = wssLn.Close()
	}
	consLock.Lock() //这段放下面，防止新连接进来触发逻辑
	for k, v := range tcpCons {
		delete(tcpCons, k)
		_ = v.Close()
	}
	for k, v := range tlsCons {
		delete(tlsCons, k)
		_ = v.Close()
	}
	consLock.Unlock() //不能使用defer解锁，防止死锁
	wgCons.Wait()
	Info("tim 已停止")
}

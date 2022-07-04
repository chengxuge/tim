package tim

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"strings"
)

var (
	httpEndOf   = []byte{0x0D, 0x0A, 0x0D, 0x0A}
	acceptFlags = []byte("Sec-WebSocket-Accept")
)

func WsBinary(p []byte) *WebFrame {
	return &WebFrame{
		IsFrameEndOf:  true,
		OpCode:        BinaryFrame,
		PayloadLength: len(p),
		PayloadData:   p,
	}
}

func WsText(txt []byte) *WebFrame {
	return &WebFrame{
		IsFrameEndOf:  true,
		OpCode:        TextFrame,
		PayloadLength: len(txt),
		PayloadData:   txt,
	}
}

func WsPing() *WebFrame {
	return &WebFrame{
		IsFrameEndOf:  true,
		OpCode:        PingFrame,
		PayloadLength: 0,
		PayloadData:   nil,
	}
}

func NewMasking() []byte {
	var buf = make([]byte, 4)
	_, _ = rand.Read(buf)
	return buf
}

func getSub(s, left, right []byte) []byte {
	if len(s) == 0 {
		return nil
	}
	var lIdx, rIdx = 0, 0
	if len(left) != 0 {
		lIdx = bytes.Index(s, left)
		if lIdx == -1 {
			return nil
		}
		lIdx += len(left)
	}
	if len(right) != 0 {
		rIdx = bytes.Index(s[lIdx:], right)
		if rIdx == -1 {
			return nil
		}
		rIdx += lIdx
	} else {
		rIdx = len(s)
	}
	return s[lIdx:rIdx]
}

func getAcceptBase64Key(base64Key string) string {
	var data = []byte(base64Key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11")
	var sha1Data = sha1.Sum(data)
	return base64.StdEncoding.EncodeToString(sha1Data[:])
}

func getRndBase64Key() string {
	var buf = make([]byte, 16)
	_, _ = rand.Read(buf)
	return base64.StdEncoding.EncodeToString(buf)
}

func serverWebSocket(reader *bytes.Buffer) (string, *WsConfig, bool) {
	var buf = reader.Bytes() //获取接收到的数据
	if idx := bytes.LastIndex(buf, httpEndOf); idx != -1 {
		var path = getSub(buf, []byte("GET "), []byte(" HTTP/1.1\r\n"))
		var host = getSub(buf, []byte("Host:"), []byte("\r\n"))
		var origin = getSub(buf, []byte("Origin:"), []byte("\r\n"))
		var protocol = getSub(buf, []byte("Sec-WebSocket-Protocol:"), []byte("\r\n"))
		var base64Key = getSub(buf, []byte("Sec-WebSocket-Key:"), []byte("\r\n"))
		if base64Key != nil {
			var b64 = string(bytes.TrimSpace(base64Key))
			var p, h, o, ptc = "", "", "", ""

			if path != nil {
				p = string(bytes.TrimSpace(path))
			}
			if host != nil {
				h = string(bytes.TrimSpace(host))
			}
			if origin != nil {
				o = string(bytes.TrimSpace(origin))
			}
			if protocol != nil {
				ptc = string(bytes.TrimSpace(protocol))
			}

			reader.Next(idx + 4) //清除已读数据

			return b64, &WsConfig{
				Path:     p,
				Host:     h,
				Origin:   o,
				Protocol: ptc,
			}, true
		}
	}
	return "", nil, false
}

func clientWebSocket(reader *bytes.Buffer) bool {
	var buf = reader.Bytes()
	if idx := bytes.LastIndex(buf, httpEndOf); idx != -1 {
		if bytes.Contains(buf, acceptFlags) {
			reader.Next(idx + 4) //清除已读数据
			return true
		}
	}
	return false
}

func getResponseOK(base64Key string, protocol string) string {
	var sb = strings.Builder{}
	sb.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	sb.WriteString("Upgrade: websocket\r\n")
	sb.WriteString("Connection: Upgrade\r\n")
	if protocol != "" {
		sb.WriteString("Sec-WebSocket-Protocol: ")
		sb.WriteString(protocol)
		sb.WriteString("\r\n")
	}
	sb.WriteString("Sec-WebSocket-Accept: ")
	sb.WriteString(getAcceptBase64Key(base64Key))
	sb.WriteString("\r\n\r\n")
	return sb.String()
}

func getRequest(wsCfg *WsConfig) string {
	var sb = strings.Builder{}
	sb.WriteString("GET ")
	if wsCfg.Path != "" {
		sb.WriteString(wsCfg.Path)
	} else {
		sb.WriteString("/")
	}
	sb.WriteString(" HTTP/1.1\r\n")
	sb.WriteString("Upgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: ")
	sb.WriteString(getRndBase64Key())
	sb.WriteString("\r\n")
	if wsCfg.Host != "" {
		sb.WriteString("Host: ")
		sb.WriteString(wsCfg.Host)
		sb.WriteString("\r\n")
	}
	if wsCfg.Origin != "" {
		sb.WriteString("Origin: ")
		sb.WriteString(wsCfg.Origin)
		sb.WriteString("\r\n")
	}
	if wsCfg.Protocol != "" {
		sb.WriteString("Sec-WebSocket-Protocol: ")
		sb.WriteString(wsCfg.Protocol)
		sb.WriteString("\r\n")
	}
	sb.WriteString("Sec-WebSocket-Version: 13\r\n\r\n")
	return sb.String()
}

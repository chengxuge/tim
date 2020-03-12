package tim

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"time"
)

func initConfig() {
	var cfgFile = "conf/server.json"
	var _, err = os.Stat(cfgFile)
	if os.IsNotExist(err) {
		err = os.MkdirAll(path.Dir(cfgFile), os.ModePerm)
		if err != nil {
			Fatal(err.Error())
		}
		file, err := os.Create(cfgFile)
		if err != nil {
			Fatal(err.Error())
		}
		data, err := json.Marshal(svrCfg)
		if err != nil {
			Fatal(err.Error())
		}
		var out = bytes.Buffer{} //换行的json格式
		_ = json.Indent(&out, data, "", "\t")
		_, _ = out.WriteTo(file)
		_ = file.Close()
	} else {
		LoadConfig(cfgFile, svrCfg)
	}
	setLogFile(svrCfg.LogRoot, svrCfg.LogLevel,
		svrCfg.LogFileMax, svrCfg.LogToStd)
	Info("初始化系统配置文件成功，正在启动")
}

func initDefaultRoute() {
	MsgRoute(&WebFrame{}, nil, func(a *Agent, msg interface{}) {
		var f = msg.(*WebFrame)
		switch f.OpCode {
		case ContinueFrame:
		//case TextFrame:
		//a.Send(WsText(f.PayloadData))
		//case BinaryFrame:
		//a.Send(WsBinary(f.PayloadData))
		case CloseFrame:
			_ = a.Conn.Close()
		case PingFrame:
			var msg = &WebFrame{
				IsFrameEndOf:  true,
				OpCode:        PongFrame,
				PayloadLength: 0,
				PayloadData:   nil,
			}
			a.Send(msg)
		case PongFrame:
		}
	})
}

func init() {
	initConfig()
	initDefaultRoute()
	if svrCfg.LRUTimeOut > 0 && svrCfg.LRUInterval > 0 {
		startLRUDetect(svrCfg.LRUTimeOut*time.Second, svrCfg.LRUInterval*time.Second)
	}
}

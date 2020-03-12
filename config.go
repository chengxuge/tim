package tim

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
)

type serverConfig struct {
	ExtendUrl    string             //扩展配置的url
	CertFile     string             //SSL文件
	KeyFile      string             //SSL_Key文件
	TcpAddr      string             //tcp监听地址
	TlsAddr      string             //ssl监听地址
	WsAddr       string             //ws监听地址
	WssAddr      string             //wss监听地址
	LogRoot      string             //日志存放目录
	LogToStd     bool               //是否输出到控制台
	LogLevel     int                //日志记录等级
	LogFileMax   int                //单文件最大记录数量
	MaxConnNum   int                //最大连接数量
	MaxReadBytes int                //最大消息长度
	BuffSize     int                //缓冲大小
	IpWhiteList  map[string]float64 //IP白名单
	MaxIpConnNum int                //单IP最大连接数
	WindowSec    int                //窗口期秒数,0为不开启
	WindowNum    int                //窗口期最大消息数量
	LRUTimeOut   time.Duration      //连接超时
	LRUInterval  time.Duration      //超时检测间隔
}

var svrCfg = &serverConfig{
	ExtendUrl:    "",
	CertFile:     "",
	KeyFile:      "",
	TcpAddr:      ":8088",
	TlsAddr:      "",
	WsAddr:       ":8800",
	WssAddr:      "",
	LogRoot:      "log/",
	LogToStd:     true,
	LogLevel:     DebugLevel,
	LogFileMax:   1024 * 1024 * 100, //默认log文件最大100m
	MaxConnNum:   10000,
	MaxReadBytes: 0xFFFF,
	BuffSize:     2048,
	IpWhiteList: map[string]float64{
		"127.0.0.1": 1000,
	},
	MaxIpConnNum: 15,
	WindowSec:    1,
	WindowNum:    5,
	LRUTimeOut:   10,
	LRUInterval:  2,
}

var ExtendConfig map[string]interface{} //扩展配置数据

func LoadConfig(file string, cfg *serverConfig) {
	var data, err = ioutil.ReadFile(file)
	if err != nil {
		Fatal(err.Error())
	}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		Fatal(err.Error())
	}
	if cfg.ExtendUrl != "" {
		var rsp, err = http.Get(cfg.ExtendUrl)
		if err != nil {
			Fatal(err.Error())
		}
		data, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			Fatal(err.Error())
		}
		defer rsp.Body.Close()
		ExtendConfig = make(map[string]interface{})
		err = json.Unmarshal(data, &ExtendConfig)
		if err != nil {
			Fatal(err.Error())
		}
		if ExtendConfig["err"].(float64) != 0 {
			Fatal("load extend config error")
		}

		for k, v := range ExtendConfig["data"].(map[string]interface{})["cfg"].(map[string]interface{}) {
			switch k {
			case "TcpAddr":
				cfg.TcpAddr = v.(string)
			case "TlsAddr":
				cfg.TlsAddr = v.(string)
			case "WsAddr":
				cfg.WsAddr = v.(string)
			case "WssAddr":
				cfg.WssAddr = v.(string)
			case "MaxConnNum":
				cfg.MaxConnNum = int(v.(float64))
			case "MaxReadBytes":
				cfg.MaxReadBytes = int(v.(float64))
			case "IpWhiteList":
				cfg.IpWhiteList = v.(map[string]float64)
			case "MaxIpConnNum":
				cfg.MaxIpConnNum = int(v.(float64))
			case "WindowSec":
				cfg.WindowSec = int(v.(float64))
			case "WindowNum":
				cfg.WindowNum = int(v.(float64))
			case "LRUTimeOut":
				cfg.LRUTimeOut = time.Duration(v.(float64))
			case "LRUInterval":
				cfg.LRUInterval = time.Duration(v.(float64))
			}
		}
	}
}

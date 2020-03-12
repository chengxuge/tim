package tim

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
)

const (
	DebugLevel = iota
	InfoLevel  = iota
	WarnLevel  = iota
	ErrorLevel = iota
)

var (
	logLevel = DebugLevel
	logger   *log.Logger
)

func setLogFile(root string, lv, max int, toStd bool) {
	var _, err = os.Stat(root)
	if os.IsNotExist(err) {
		err := os.MkdirAll(root, os.ModePerm)
		if err != nil {
			Fatal(err.Error())
		}
	}
	file, err := newLogFile(root, max)
	if err != nil {
		Fatal(err.Error())
		return
	}
	if !toStd {
		logger = log.New(file, "", log.LstdFlags)
	} else {
		logger = log.New(io.MultiWriter(file, os.Stderr), "", log.LstdFlags)
	}
	logLevel = lv //设置日志等级
}

func Debug(format string, v ...interface{}) {
	if DebugLevel >= logLevel {
		_ = logger.Output(2, fmt.Sprintf("[Debug]"+format, v...))
	}
}

func Info(format string, v ...interface{}) {
	if InfoLevel >= logLevel {
		_ = logger.Output(2, fmt.Sprintf("[Info]"+format, v...))
	}
}

func Warn(format string, v ...interface{}) {
	if WarnLevel >= logLevel {
		_ = logger.Output(2, fmt.Sprintf("[Warn]"+format, v...))
	}
}

func Error(format string, v ...interface{}) {
	if ErrorLevel >= logLevel {
		_ = logger.Output(2, fmt.Sprintf("[Error]"+format, v...))
	}
}

func Fatal(format string, v ...interface{}) {
	_ = logger.Output(2, fmt.Sprintf("[Fatal]"+format, v...)) //输出到日志文件
	fmt.Println(fmt.Sprintf("[Fatal]"+format, v...))          //输出到控制台
	os.Exit(1)
}

func GetPanicStack() string {
	s := []byte("/src/runtime/panic.go")
	e := []byte("\ngoroutine ")
	line := []byte("\n")
	stack := make([]byte, 8192) //8KB
	length := runtime.Stack(stack, false)
	start := bytes.Index(stack, s)
	stack = stack[start:length]
	start = bytes.Index(stack, line)
	stack = stack[start+1:]
	end := bytes.LastIndex(stack, line)
	if end != -1 {
		stack = stack[:end]
	}
	end = bytes.Index(stack, e)
	if end != -1 {
		stack = stack[:end]
	}
	stack = bytes.TrimRight(stack, "\n")
	return string(stack)
}

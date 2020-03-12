package tim

import (
	"fmt"
	"os"
	"time"
)

type logFile struct {
	file     *os.File
	root     string
	maxWrite int
	curWrite int
}

func (w *logFile) Write(b []byte) (n int, err error) {
	n, err = w.file.Write(b)
	w.curWrite += n
	if w.curWrite >= w.maxWrite {
		var oldFile = w.file //关闭原来的log文件
		if w.createFile(w.root) == nil {
			_ = oldFile.Close()
			w.curWrite = 0
		}
	}
	return
}

func (w *logFile) createFile(root string) error {
	var now = time.Now()
	var file, err = os.Create(fmt.Sprintf("%s%d%02d%02d_%02d%02d%02d.log",
		root,
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour(),
		now.Minute(),
		now.Second()))
	if err != nil {
		return err
	}
	w.file = file
	return nil
}

func newLogFile(root string, maxWrite int) (*logFile, error) {
	var logFile = &logFile{
		file:     nil,
		root:     root,
		maxWrite: maxWrite,
		curWrite: 0,
	}
	if err := logFile.createFile(root); err != nil {
		return nil, err
	}
	return logFile, nil
}

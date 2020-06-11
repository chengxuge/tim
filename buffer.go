package tim

import (
	"bytes"
	"unsafe"
)

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
	ptr.buf, ptr.off = ptr.buf[:len(buf)], 0
	copy(ptr.buf, buf)
}

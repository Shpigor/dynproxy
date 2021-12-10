package dynproxy

import (
	"net"
)

type Session interface {
	Init(buffer []byte) error
	//
	ProcessRead(fd int, buffer []byte) error
	//
	GetConnByFd(fd int) net.Conn
	//
	GetFds() []int
	//
	Close() error
	//
}

func generateId(src, dst net.Conn) string {
	return src.RemoteAddr().String() + "<->" + dst.RemoteAddr().String()
}

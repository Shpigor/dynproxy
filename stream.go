package dynproxy

import (
	"net"
)

type Direction int8

const (
	From = Direction(0)
	To   = Direction(1)
)

type Stream interface {
	//
	ProcessRead(dir Direction, buffer []byte) error
	//
	GetConnByDirection(dir Direction) net.Conn
	//
	GetFd(dir Direction) int
	//
	GetFds() []int
	//
	Close() error
	//
	//Init() error
}

func generateId(src, dst net.Conn) string {
	return src.RemoteAddr().String() + "<->" + dst.RemoteAddr().String()
}

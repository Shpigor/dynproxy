package dynproxy

import "net"

type Event struct {
	id    string
	Type  int
	key   string
	value string
}

type newConn struct {
	frontend net.Conn
	backend  string
}

type status struct {
	Name   string
	Status int
}

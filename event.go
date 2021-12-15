package dynproxy

import (
	"net"
	"time"
)

type Event struct {
	Id        string
	Timestamp int64
	Type      int
	MetaData  map[string]interface{}
	Tags      []string
	Err       error
	Msg       string
}

func genOcspErrorEvent(id string, err error, msg string) Event {
	return Event{
		Id:        id,
		Timestamp: time.Now().UnixMilli(),
		Err:       err,
		Msg:       msg,
	}
}

type newConn struct {
	frontend net.Conn
	backend  string
}

type status struct {
	Name   string
	Status int
}

package dynproxy

import (
	"net"
	"time"
)

const (
	OcspValidationError           = 500
	UnavailableOcspResponderError = 503
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

func genOcspErrorEvent(id string, errorType int, err error, msg string) Event {
	return Event{
		Id:        id,
		Timestamp: time.Now().UnixMilli(),
		Type:      errorType,
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

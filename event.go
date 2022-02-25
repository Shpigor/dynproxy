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
	Id        string                 `json:"id"`
	Timestamp int64                  `json:"timestamp"`
	Type      int                    `json:"type"`
	MetaData  map[string]interface{} `json:"metaData"`
	Tags      []string               `json:"tags"`
	Err       error                  `json:"error"`
	Msg       string                 `json:"msg"`
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

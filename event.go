package dynproxy

import (
	"net"
	"time"
)

const (
	NewConn EventType = iota
	BackendStatus
	Monitor
	OcspValidationError
	UnavailableOcspResponderError
)

type EventType int

type Event struct {
	id           string
	Timestamp    int64
	Type         EventType
	Data         map[string]string
	MonitorEvent *MonitorEvent
	ErrorEvent   *ErrorEvent
	NewConnEvent *NewConnEvent
}

type MonitorEvent struct {
	MetaData map[string]interface{}
	Tags     []string
}

type ErrorEvent struct {
	Err error
	Msg string
}

type NewConnEvent struct {
	frontConn net.Conn
	backends  string
}

func (e *Event) Id() uint64 {
	return 0
}

func (e *Event) IsNewConn() bool {
	return e.Type == NewConn
}

func (e *Event) GetNewConn() *NewConnEvent {
	return e.NewConnEvent
}

func (e Event) IsBackendStatus() bool {
	return e.Type == BackendStatus
}

func buildOcspErrorEvent(id string, errorType EventType, err error, msg string) *Event {
	return &Event{
		id:        id,
		Timestamp: time.Now().UnixMilli(),
		Type:      errorType,
		ErrorEvent: &ErrorEvent{
			Err: err,
			Msg: msg,
		},
	}
}

func buildNewConnEvent(frontConn net.Conn, backends string) *Event {
	return &Event{
		Timestamp: time.Now().UnixMilli(),
		Type:      NewConn,
		NewConnEvent: &NewConnEvent{
			frontConn: frontConn,
			backends:  backends,
		},
	}
}

type status struct {
	Name   string
	Status int
}

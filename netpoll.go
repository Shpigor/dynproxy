package dynproxy

import (
	"golang.org/x/sys/unix"
)

const (
	defEventsBufferSize = 128
	blocked             = -1
	nonBlocked          = 0
)

type PollerConfig struct {
	EventBufferSize int
	EventQueueSize  int
}

type Poller struct {
	eventBufferSize int
	eventQueueSize  int
	fd              int
	lockOSThread    bool
	events          []unix.EpollEvent
	timeout         int
}

type SocketEvent struct {
	Events uint32
	Fd     int32
}

type PollDesc struct {
	FD int
}

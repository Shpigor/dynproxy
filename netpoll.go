package dynproxy

const (
	defEventsBufferSize = 128
	blocked             = -1
	nonBlocked          = 0
)

type PollerConfig struct {
	EventBufferSize int
	EventQueueSize  int
}

type Poller interface {
	WaitForEvents(handler NetEventHandler, holder SessionHolder) (int, error)
	AddRead(fd int) error
	AddReadErrors(fd int) error
	DeletePoll(fd int) error
	Close()
}

type SocketEvent struct {
	Events uint32
	Fd     int32
}

type PollDesc struct {
	FD int
}

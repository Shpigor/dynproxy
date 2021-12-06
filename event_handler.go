package dynproxy

type EventHandler interface {
	// ReadEvent Handle read events received from polling
	ReadEvent(stream *Stream, direction int) error
	// WriteEvent Handle write events received from polling
	WriteEvent(stream *Stream, direction int) error
	// ErrorEvent Handle error events received from polling
	ErrorEvent(stream *Stream, direction int, errors []error) error
}

type StreamProvider interface {
	FindStreamByFd(fd int) (*Stream, int)
	AddStream(fd int, p *Stream)
	RemoveByFd(fd int)
	RemoveStream(stream *Stream)
}

package dynproxy

import (
	"github.com/rs/zerolog/log"
	"sync"
)

type EventHandler interface {
	// ReadEvent Handle read events received from polling
	ReadEvent(stream Stream, dir Direction) error
	// ErrorEvent Handle error events received from polling
	ErrorEvent(stream Stream, errors []error) error
}

type StreamProvider interface {
	FindStreamByFd(fd int) (Stream, Direction)
	AddStream(stream Stream)
	RemoveStream(stream Stream)
}

func NewBufferHandler() EventHandler {
	return &bufferHandler{
		bb: make([]byte, 4096),
	}
}

type bufferHandler struct {
	bb []byte
}

func (h *bufferHandler) ReadEvent(stream Stream, dir Direction) error {
	if stream == nil {
		return noStreamFound
	}
	return stream.ProcessRead(dir, h.bb)
}

func (h *bufferHandler) ErrorEvent(stream Stream, errors []error) error {
	if stream != nil {
		err := stream.Close()
		if err != nil {
			log.Error().Msgf("got error while close stream: %+v", err)
			return err
		}
	}
	return closedStream
}

func NewMapStreamProvider() StreamProvider {
	return &mapStreamProvider{
		lock:    &sync.RWMutex{},
		streams: make(map[int]Stream),
	}
}

type mapStreamProvider struct {
	lock    *sync.RWMutex
	streams map[int]Stream
}

func (sp *mapStreamProvider) FindStreamByFd(fd int) (Stream, Direction) {
	sp.lock.RLock()
	defer sp.lock.RUnlock()
	stream, ok := sp.streams[fd]
	if ok {
		if fd == stream.GetFd(From) {
			return stream, From
		}
		return stream, To
	}
	return nil, 0
}

func (sp *mapStreamProvider) AddStream(stream Stream) {
	sp.lock.Lock()
	defer sp.lock.Unlock()
	fds := stream.GetFds()
	for _, fd := range fds {
		sp.streams[fd] = stream
	}
}

func (sp *mapStreamProvider) RemoveStream(stream Stream) {
	sp.lock.Lock()
	defer sp.lock.Unlock()
	fds := stream.GetFds()
	for _, fd := range fds {
		delete(sp.streams, fd)
	}
}

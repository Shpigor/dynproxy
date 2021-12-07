package dynproxy

import (
	"github.com/rs/zerolog/log"
	"net"
)

type proxyStream struct {
	id        string
	srvFd     int
	frontFd   int
	backend   net.Conn
	frontend  net.Conn
	eventChan chan Event
	handler   func(src, dst net.Conn, data []byte) error
}

func NewDefaultProxyStream(frontConn net.Conn, srvConn net.Conn, eventChan chan Event) (Stream, error) {
	return NewProxyStream(frontConn, srvConn, eventChan, copyFromSrcToDst)
}

func NewProxyStream(frontConn net.Conn, srvConn net.Conn, eventChan chan Event, handler func(src, dst net.Conn, data []byte) error) (Stream, error) {
	frontFd, err := ConnToFileDesc(frontConn)
	if err != nil {
		return nil, err
	}
	srvFd, err := ConnToFileDesc(srvConn)
	if err != nil {
		return nil, err
	}
	return &proxyStream{
		id:        generateId(frontConn, srvConn),
		frontFd:   int(frontFd),
		frontend:  frontConn,
		srvFd:     int(srvFd),
		backend:   srvConn,
		eventChan: eventChan,
		handler:   handler,
	}, nil
}

func (s *proxyStream) ProcessRead(dir Direction, buffer []byte) error {
	if dir == From {
		if log.Debug().Enabled() {
			log.Debug().Msgf("[%d] read event from stream: %s", s.frontFd, s.id)
		}
		return s.handler(s.frontend, s.backend, buffer)
	} else {
		if log.Debug().Enabled() {
			log.Debug().Msgf("[%d] read event to stream: %s", s.srvFd, s.id)
		}
		return s.handler(s.backend, s.frontend, buffer)
	}
}

func (s *proxyStream) GetConnByDirection(dir Direction) net.Conn {
	if dir == From {
		return s.frontend
	} else {
		return s.backend
	}
}

func (s *proxyStream) Close() error {
	err := s.frontend.Close()
	if s.backend != nil {
		err = s.backend.Close()
	}
	return err
}

func (s *proxyStream) GetFd(dir Direction) int {
	if dir == From {
		return s.frontFd
	}
	return s.srvFd
}

func (s *proxyStream) GetFds() []int {
	return []int{s.frontFd, s.srvFd}
}

func (s *proxyStream) Init() {

}

func copyFromSrcToDst(src, dst net.Conn, buffer []byte) error {
	read, err := src.Read(buffer)
	if err != nil {
		log.Printf("got error while reading data from frontend: %+v", err)
		return err
	}
	if read > 0 {
		_, err := dst.Write(buffer[:read])
		if err != nil {
			log.Printf("got error while writing data to backend: %+v", err)
			return err
		}
	}
	return nil
}

package dynproxy

import (
	"github.com/rs/zerolog/log"
	"net"
)

type proxySession struct {
	id        string
	backendFd int
	frontFd   int
	backend   net.Conn
	frontend  net.Conn
	eventChan chan Event
	handler   func(src, dst net.Conn, data []byte) error
}

func NewDefaultProxySession(frontConn net.Conn, srvConn net.Conn, eventChan chan Event) (Session, error) {
	return NewProxySession(frontConn, srvConn, eventChan, copyFromSrcToDst)
}

func NewProxySession(frontConn net.Conn, backendConn net.Conn, eventChan chan Event, handler func(src, dst net.Conn, data []byte) error) (Session, error) {
	frontFd, _, err := ConnToFileDesc(frontConn)
	if err != nil {
		return nil, err
	}
	backendFd, _, err := ConnToFileDesc(backendConn)
	if err != nil {
		return nil, err
	}
	return &proxySession{
		id:        generateId(frontConn, backendConn),
		frontFd:   frontFd,
		frontend:  frontConn,
		backendFd: backendFd,
		backend:   backendConn,
		eventChan: eventChan,
		handler:   handler,
	}, nil
}

func (s *proxySession) Init(buffer []byte) error {
	err := s.ProcessRead(s.frontFd, buffer)
	if err != nil {
		return err
	}
	err = s.ProcessRead(s.backendFd, buffer)
	if err != nil {
		return err
	}
	return nil
}

func (s *proxySession) ProcessRead(fd int, buffer []byte) error {
	if fd == s.frontFd {
		if log.Debug().Enabled() {
			log.Debug().Msgf("[%d] read event from stream: %s", s.frontFd, s.frontend.RemoteAddr().String())
		}
		return s.handler(s.frontend, s.backend, buffer)
	} else {
		if log.Debug().Enabled() {
			log.Debug().Msgf("[%d] read event to stream: %s", s.backendFd, s.backend.RemoteAddr().String())
		}
		return s.handler(s.backend, s.frontend, buffer)
	}
}

func (s *proxySession) GetConnByFd(fd int) net.Conn {
	if fd == s.frontFd {
		return s.frontend
	}
	return s.backend
}

func (s *proxySession) Close() error {
	err := s.frontend.Close()
	if s.backend != nil {
		err = s.backend.Close()
	}
	return err
}

func (s *proxySession) GetFds() []int {
	return []int{s.frontFd, s.backendFd}
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

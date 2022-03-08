package dynproxy

import (
	"github.com/rs/zerolog/log"
	"net"
	"time"
)

type proxySession struct {
	id        string
	backendFd int
	frontFd   int
	backend   net.Conn
	frontend  net.Conn
	eventChan chan *Event
	stats     *proxySessionStats
}

type proxySessionStats struct {
	LastActivityTime   int64
	TotalSentBytes     uint64
	TotalReceivedBytes uint64
}

func NewDefaultProxySession(frontConn net.Conn, srvConn net.Conn, eventChan chan *Event) (Session, error) {
	return NewProxySession(frontConn, srvConn, eventChan)
}

func NewProxySession(frontConn net.Conn, backendConn net.Conn, eventChan chan *Event) (Session, error) {
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
		stats:     &proxySessionStats{},
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
		return s.copyFromFrontend(buffer)
	} else {
		return s.copyFromBackend(buffer)
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
	if err != nil {
		log.Debug().Msgf("closed frontend session error: %+v", err)
	}
	if s.backend != nil {
		err = s.backend.Close()
		if err != nil {
			log.Debug().Msgf("closed backend session error: %+v", err)
		}
	}
	if log.Debug().Enabled() {
		log.Debug().Msgf("closed session: %+v", s)
	}
	return err
}

func (s *proxySession) GetFds() []int {
	return []int{s.frontFd, s.backendFd}
}

func (s *proxySession) GetId() string {
	return s.id
}

func (s *proxySession) GetStats() SessionStats {
	return SessionStats{
		LastActivityTime:   s.stats.LastActivityTime,
		TotalSentBytes:     s.stats.TotalSentBytes,
		TotalReceivedBytes: s.stats.TotalReceivedBytes,
	}
}

func (s *proxySession) copyFromFrontend(buffer []byte) error {
	read, err := s.frontend.Read(buffer)
	if err != nil {
		log.Printf("got error while reading data from:%+v, error: %+v", s.frontend.RemoteAddr(), err)
		return err
	}
	if read > 0 {
		write, err := s.backend.Write(buffer[:read])
		if err != nil {
			log.Printf("got error while writing data to:%+v error: %+v", s.backend.RemoteAddr(), err)
			return err
		}
		if log.Debug().Enabled() {
			log.Debug().Msgf("read %d bytes from: %s and write %d bytes to %s", read, s.frontend.RemoteAddr().String(), write, s.backend.RemoteAddr().String())
		}
		s.stats.LastActivityTime = time.Now().UnixMilli()
		s.stats.TotalReceivedBytes += uint64(read)
	}
	return nil
}

func (s *proxySession) copyFromBackend(buffer []byte) error {
	read, err := s.backend.Read(buffer)
	if err != nil {
		log.Printf("got error while reading data from:%+v, error: %+v", s.backend.RemoteAddr(), err)
		return err
	}
	if read > 0 {
		write, err := s.frontend.Write(buffer[:read])
		if err != nil {
			log.Printf("got error while writing data to:%+v error: %+v", s.frontend.RemoteAddr(), err)
			return err
		}
		if log.Debug().Enabled() {
			log.Debug().Msgf("read %d bytes from: %s and write %d bytes to %s", read, s.backend.RemoteAddr().String(), write, s.frontend.RemoteAddr().String())
		}
		s.stats.LastActivityTime = time.Now().UnixMilli()
		s.stats.TotalSentBytes += uint64(write)
	}
	return nil
}

package dynproxy

import (
	"github.com/rs/zerolog/log"
	"net"
)

type clientSession struct {
	id        string
	fd        int
	conn      net.Conn
	eventChan chan Event
	handler   func(src, dst net.Conn, data []byte) error
}

func NewEchoClientSession(conn net.Conn, eventChan chan Event) (Session, error) {
	return NewClientSession(conn, eventChan, echo)
}

func NewClientSession(conn net.Conn, eventChan chan Event, handler func(src, dst net.Conn, data []byte) error) (Session, error) {
	fd, _, err := ConnToFileDesc(conn)
	if err != nil {
		return nil, err
	}
	return &clientSession{
		id:        generateId(conn, conn),
		fd:        fd,
		conn:      conn,
		eventChan: eventChan,
		handler:   handler,
	}, nil
}
func (s *clientSession) Init(buffer []byte) error {
	return s.ProcessRead(s.fd, buffer)
}

func (s *clientSession) ProcessRead(fd int, buffer []byte) error {
	if log.Debug().Enabled() {
		log.Debug().Msgf("[%d] read event from stream: %s", s.fd, s.id)
	}
	return s.handler(s.conn, s.conn, buffer)
}

func (s *clientSession) GetConnByFd(fd int) net.Conn {
	return s.conn
}

func (s *clientSession) Close() error {
	return s.conn.Close()
}

func (s *clientSession) GetFds() []int {
	return []int{s.fd}
}

func (s *clientSession) GetId() string {
	return s.id
}

func (s *clientSession) GetStats() SessionStats {
	return SessionStats{}
}

func echo(src, dst net.Conn, buffer []byte) error {
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

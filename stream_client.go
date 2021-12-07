package dynproxy

import (
	"github.com/rs/zerolog/log"
	"net"
)

type clientStream struct {
	id        string
	fd        int
	conn      net.Conn
	eventChan chan Event
	handler   func(src, dst net.Conn, data []byte) error
}

func NewEchoClientStream(conn net.Conn, eventChan chan Event) (Stream, error) {
	return NewClientStream(conn, eventChan, echo)
}

func NewClientStream(conn net.Conn, eventChan chan Event, handler func(src, dst net.Conn, data []byte) error) (Stream, error) {
	fd, err := ConnToFileDesc(conn)
	if err != nil {
		return nil, err
	}
	return &clientStream{
		id:        generateId(conn, conn),
		fd:        int(fd),
		conn:      conn,
		eventChan: eventChan,
		handler:   handler,
	}, nil
}

func (s *clientStream) ProcessRead(dir Direction, buffer []byte) error {
	if log.Debug().Enabled() {
		log.Debug().Msgf("[%d] read event from stream: %s", s.fd, s.id)
	}
	return s.handler(s.conn, s.conn, buffer)
}

func (s *clientStream) GetConnByDirection(dir Direction) net.Conn {
	return s.conn
}

func (s *clientStream) Close() error {
	return s.conn.Close()
}

func (s *clientStream) GetFd(dir Direction) int {
	if dir == From {
		return s.fd
	}
	return -1
}

func (s *clientStream) GetFds() []int {
	return []int{s.fd}
}

func (s *clientStream) Init() error {
	//s.
	return nil
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

package dynproxy

import (
	"log"
	"net"
)

type Stream struct {
	id       string
	backend  net.Conn
	frontend net.Conn
	finish   chan string
}

// Start TODO: need to use epoll implementation
func (s *Stream) start() {
	// TODO: need to make it dynamic
	buffer := make([]byte, 1024)
	for {
		err := readWrite(s.frontend, s.backend, buffer)
		if err != nil {
			s.finish <- s.id
			return
		}
		err = readWrite(s.backend, s.frontend, buffer)
		if err != nil {
			s.finish <- s.id
			return
		}
	}
}

func readWrite(src net.Conn, dst net.Conn, buffer []byte) error {
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

func (s *Stream) ReadWrite(direction int, buffer []byte) error {
	var src net.Conn
	var dst net.Conn
	if direction == 0 {
		src = s.frontend
		dst = s.backend
	} else {
		src = s.backend
		dst = s.frontend
	}
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

func (s *Stream) GetConnByDirection(direction int) net.Conn {
	if direction == 0 {
		return s.frontend
	} else {
		return s.backend
	}
}

func (s *Stream) Close(direction int) error {
	s.frontend.Close()
	s.backend.Close()
	return nil
}
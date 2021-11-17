package dynproxy

import (
	"errors"
	"io"
	"log"
	"net"
)

type PipeManager struct {
	Pipes map[string]*Pipe
}

type Pipe struct {
	backend  net.Conn
	frontend net.Conn
}

// Start TODO: need to use epoll implementation
func (p *Pipe) Start() {
	// TODO: need to make it dynamic
	buffer := make([]byte, 1024)
	for {
		err := readWrite(p.frontend, p.backend, buffer)
		if err != nil {
			// TODO: close sessions
			return
		}
		err = readWrite(p.backend, p.frontend, buffer)
		if err != nil {
			// TODO: close sessions
			return
		}
	}
}

func readWrite(src net.Conn, dst net.Conn, buffer []byte) error {
	read, err := src.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		log.Printf("got error while reading data from frontend")
		return err
	}
	if read > 0 {
		_, err := dst.Write(buffer[:read])
		if err != nil {
			log.Printf("got error while writing data to backend")
			return err
		}
	}
	return nil
}

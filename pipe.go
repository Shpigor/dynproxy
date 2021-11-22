package dynproxy

import (
	"log"
	"net"
)

type pipe struct {
	id       string
	backend  net.Conn
	frontend net.Conn
	finish   chan string
}

// Start TODO: need to use epoll implementation
func (p *pipe) start() {
	// TODO: need to make it dynamic
	buffer := make([]byte, 1024)
	for {
		err := readWrite(p.frontend, p.backend, buffer)
		if err != nil {
			p.finish <- p.id
			return
		}
		err = readWrite(p.backend, p.frontend, buffer)
		if err != nil {
			p.finish <- p.id
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

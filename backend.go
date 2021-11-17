package dynproxy

import (
	"context"
	"github.com/rs/zerolog/log"
	"net"
	"time"
)

type Backend struct {
	ctx         context.Context
	Name        string
	Address     string
	Net         string
	Status      int
	Conn        net.Conn
	HealthCheck *HealthCheck
}

type HealthCheck struct {
	Period int
}

func (b *Backend) Start() error {
	conn, err := net.Dial("tcp", b.Address)
	if err != nil {
		return err
	}
	b.Conn = conn
	if b.HealthCheck != nil {
		go b.runHealthCheck()
	}
	return nil
}

func (b *Backend) runHealthCheck() {
	ticker := time.NewTicker(time.Duration(b.HealthCheck.Period) * time.Second)
	for {
		select {
		case <-b.ctx.Done():
			log.Printf("stopped health check for backend: %s", b.Name)
			ticker.Stop()
			err := b.Conn.Close()
			if err != nil {
				log.Printf("got error while closing connection: %+v", err)
			}
			return
		case <-ticker.C:
			if b.Conn != nil {
				tcpConn, ok := b.Conn.(*net.TCPConn)
				if ok {
					log.Printf("healshcheck for backend: %+v", tcpConn)
				}
			}
		}
	}

}

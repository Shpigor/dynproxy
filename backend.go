package dynproxy

import (
	"context"
	"github.com/rs/zerolog/log"
	"net"
	"time"
)

const (
	unknown  = -1
	enabled  = 1
	disabled = 0
)

type Backend struct {
	ctx           context.Context
	Name          string
	Address       string
	Net           string
	Status        int
	HealthCheck   *HealthCheck
	checkBuf      []byte
	updateChannel chan status
}

type HealthCheck struct {
	Period int
}

func (b *Backend) initBackend() {
	log.Info().Msgf("starting backend: %s %s ...", b.Name, b.Address)
	b.Status = unknown
	b.checkBuf = make([]byte, 1)
	b.updateChannel = b.ctx.Value("channel").(chan status)
	if b.HealthCheck != nil {
		go b.runHealthCheck()
	} else {
		b.Status = enabled
	}
}

func (b *Backend) runHealthCheck() {
	log.Debug().Msgf("running health check for backend: %+s ...", b.Name)
	ticker := time.NewTicker(time.Duration(b.HealthCheck.Period) * time.Second)
	for {
		select {
		case <-b.ctx.Done():
			log.Info().Msgf("stopped health check for backend: %s", b.Name)
			ticker.Stop()
			return
		case <-ticker.C:
			b.checkConnection()
		}
	}
}

func (b *Backend) getBackendConn() (net.Conn, error) {
	if b.Status != disabled {
		return net.Dial(b.Net, b.Address)
	}
	return nil, noActiveBackends
}

func (b *Backend) checkConnection() {
	state := enabled
	conn, err := net.Dial(b.Net, b.Address)
	if err != nil {
		log.Debug().Msgf("got error while connecting to backend: %+v", err)
		state = disabled
	} else {
		_, err = conn.Read(b.checkBuf)
		if err != nil {
			log.Debug().Msgf("got error while connecting to backend: %+v", err)
			state = disabled
		}
		err = conn.Close()
		if err != nil {
			log.Debug().Msgf("got error while connecting to backend: %+v", err)
		}
	}
	if b.Status != state {
		if b.updateChannel != nil {
			b.updateChannel <- status{b.Name, state}
		}
		b.Status = state
	}
}

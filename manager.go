package dynproxy

import (
	"context"
	"github.com/rs/zerolog/log"
	"net"
)

type ContextManager struct {
	ctx          context.Context
	Pipes        map[string]*Stream
	newFrontConn chan *newConn
	pipeEnd      chan string
}

type newConn struct {
	frontend net.Conn
	backend  string
}

func NewContextManager(ctx context.Context) *ContextManager {
	cm := &ContextManager{
		ctx:          ctx,
		Pipes:        make(map[string]*Stream),
		newFrontConn: make(chan *newConn, 100),
		pipeEnd:      make(chan string, 100),
	}
	go cm.start()
	return cm
}

func (cm *ContextManager) InitFrontends(config Config) {
	for _, frConfig := range config.Frontends {
		frCtx := context.WithValue(cm.ctx, "name", frConfig.Name)
		frontend := Frontend{
			Context:         frCtx,
			Net:             frConfig.Net,
			Address:         frConfig.Address,
			Name:            frConfig.Name,
			connChannel:     cm.newFrontConn,
			defaultBalancer: frConfig.BackendGroup,
			ocspProc:        NewOcspProcessor(context.WithValue(frCtx, "name", "OCSP"), frConfig),
			TlsConfig: &TlsConfig{
				SkipVerify: frConfig.TlsSkipVerify,
				CACertPath: frConfig.TlsCACertPath,
				CertPath:   frConfig.TlsCertPath,
				PkPath:     frConfig.TlsPkPath},
		}
		frontend.Listen()
	}
}

func (cm *ContextManager) start() {
	for {
		select {
		case <-cm.ctx.Done():
			return
		case newConn := <-cm.newFrontConn:
			backendConn, err := getConnByBalancerName(newConn.backend)
			if err != nil {
				log.Warn().Msgf("can't create any new connections to the backends: %+v", err)
			} else {
				id := newConn.frontend.RemoteAddr().String() + "<->" + backendConn.RemoteAddr().String()
				newPipe := &Stream{
					id:       id,
					frontend: newConn.frontend,
					backend:  backendConn,
					finish:   cm.pipeEnd,
				}
				log.Debug().Msgf("new stream: %s", id)
				cm.Pipes[id] = newPipe
				go newPipe.start()
			}
		case id := <-cm.pipeEnd:
			log.Debug().Msgf("stream: %s closed", id)
			delete(cm.Pipes, id)
		}
	}
}

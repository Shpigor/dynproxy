package dynproxy

import (
	"context"
	"github.com/rs/zerolog/log"
)

type ContextManager struct {
	ctx           context.Context
	sessionHolder SessionHolder
	handler       NetEventHandler
	newFrontConn  chan *newConn
	events        chan Event
	eventLoops    *EventLoop
}

func NewContextManager(ctx context.Context) *ContextManager {
	eventLoop, err := NewEventLoop(EventLoopConfig{
		Name:            "MainLoop",
		EventBufferSize: 256,
		LockOsThread:    true,
	})
	if err != nil {
		log.Fatal().Msgf("can't init event loop: %+v", err)
	}
	cm := &ContextManager{
		ctx:           ctx,
		sessionHolder: NewMapSessionProvider(context.WithValue(ctx, "name", "session holder")),
		handler:       NewBufferHandler(),
		newFrontConn:  make(chan *newConn, 256),
		events:        make(chan Event, 128),
		eventLoops:    eventLoop,
	}
	go cm.start()
	go eventLoop.Start(cm.handler, cm.sessionHolder)
	return cm
}

func (cm *ContextManager) InitFrontends(config Config) {
	//processor := NewOcspProcessor(context.WithValue(cm.ctx, "name", "OCSP"), frConfig, cm.events)
	for _, frConfig := range config.Frontends {
		frCtx := context.WithValue(cm.ctx, "name", frConfig.Name)
		frontend := Frontend{
			Context:         frCtx,
			Net:             frConfig.Net,
			Address:         frConfig.Address,
			Name:            frConfig.Name,
			connChannel:     cm.newFrontConn,
			defaultBalancer: frConfig.BackendGroup,
			ocspProc:        NewOcspProcessor(context.WithValue(cm.ctx, "name", "OCSP"), frConfig, cm.events),
			TlsConfig: &TlsConfig{
				SkipVerify: frConfig.TlsSkipVerify,
				CACertPath: frConfig.TlsCACertPath,
				CertPath:   frConfig.TlsCertPath,
				PkPath:     frConfig.TlsPkPath},
		}
		err := frontend.Listen()
		if err != nil {
			log.Error().Msgf("error occurred when listening frontend socket:%+v", err)
		}
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
				session, err := NewDefaultProxySession(newConn.frontend, backendConn, cm.events)
				if err != nil {
					log.Debug().Msgf("new session: %s", session)
					continue
				}
				cm.sessionHolder.AddSession(session)
				err = cm.eventLoops.PollForReadAndErrors(session.GetFds()...)
				if err != nil {
					log.Error().Msgf("got error while attach read netpoll: %+v", err)
				}
				session.Init(cm.handler.GetBuffer())
			}
		case event := <-cm.events:
			log.Debug().Msgf("received event: %+v", event)
			if event.Type == OcspValidationError {
				//cm.sessionHolder.RemoveSession()
			}
		}
	}
}

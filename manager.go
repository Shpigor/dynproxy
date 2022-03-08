package dynproxy

import (
	"context"
	"github.com/rs/zerolog/log"
)

type ContextManager struct {
	ctx           context.Context
	sessionHolder SessionHolder
	handler       NetEventHandler
	events        chan *Event
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
		events:        make(chan *Event, 256),
		eventLoops:    eventLoop,
	}
	go cm.start()
	go eventLoop.Start(cm.handler, cm.sessionHolder)
	return cm
}

func (cm *ContextManager) GetEventChannel() chan *Event {
	return cm.events
}

func (cm *ContextManager) start() {
	for {
		select {
		case <-cm.ctx.Done():
			return
		case event := <-cm.events:
			log.Debug().Msgf("received event: %+v", event)
			if event.IsNewConn() {
				newConn := event.GetNewConn()
				backendConn, err := getConnByBalancerName(newConn.backends, event)
				if err != nil {
					log.Warn().Msgf("can't create any new connections to the backends: %+v", err)
				} else {
					session, err := NewDefaultProxySession(newConn.frontConn, backendConn, cm.events)
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
			} else if event.Type == OcspValidationError {
				//cm.sessionHolder.RemoveSession()
			}
		}
	}
}

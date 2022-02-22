package dynproxy

import (
	"context"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
)

type NetEventHandler interface {
	// ReadEvent Handle read events received from polling
	ReadEvent(session Session, fd int) error
	// ErrorEvent Handle error events received from polling
	ErrorEvent(session Session, errors []error) error

	GetBuffer() []byte
}

type SessionHolder interface {
	FindSessionByFd(fd int) (Session, error)
	AddSession(session Session)
	RemoveSession(session Session)
}

func NewBufferHandler() NetEventHandler {
	return &bufferHandler{
		bb: make([]byte, 4096),
	}
}

type bufferHandler struct {
	bb []byte
}

func (h *bufferHandler) ReadEvent(session Session, fd int) error {
	if session == nil {
		return noSessionFound
	}
	return session.ProcessRead(fd, h.bb)
}

func (h *bufferHandler) ErrorEvent(session Session, errors []error) error {
	if session != nil {
		err := session.Close()
		if err != nil {
			log.Error().Msgf("got error while close session: %+v", err)
		}
	}
	return closedSession
}
func (h *bufferHandler) GetBuffer() []byte {
	return h.bb
}

func NewMapSessionProvider(ctx context.Context) SessionHolder {
	sessionHolder := &mapSessionHolder{
		ctx:      ctx,
		lock:     &sync.RWMutex{},
		sessions: make(map[int]Session),
	}
	go sessionHolder.init()
	return sessionHolder
}

type mapSessionHolder struct {
	ctx      context.Context
	lock     *sync.RWMutex
	sessions map[int]Session
}

func (sp *mapSessionHolder) FindSessionByFd(fd int) (Session, error) {
	sp.lock.RLock()
	defer sp.lock.RUnlock()
	session, ok := sp.sessions[fd]
	if ok {
		return session, nil
	}
	return nil, noSessionFound
}

func (sp *mapSessionHolder) AddSession(session Session) {
	sp.lock.Lock()
	defer sp.lock.Unlock()
	fds := session.GetFds()
	for _, fd := range fds {
		sp.sessions[fd] = session
	}
}

func (sp *mapSessionHolder) RemoveSession(session Session) {
	sp.lock.Lock()
	defer sp.lock.Unlock()
	fds := session.GetFds()
	for _, fd := range fds {
		delete(sp.sessions, fd)
	}
}

func (sp *mapSessionHolder) init() {
	ticker := time.NewTicker(20 * time.Second)
	for {
		select {
		case <-sp.ctx.Done():
			return
		case <-ticker.C:
			sp.lock.RLock()
			log.Debug().Msgf("Total sessions: %d", len(sp.sessions))
			for _, session := range sp.sessions {
				fds := session.GetFds()
				stats := session.GetStats()
				log.Debug().Msgf("%v session:[%s] lastActiveTime: %d sent: %d received: %d", fds, session.GetId(), stats.LastActivityTime, stats.TotalSentBytes, stats.TotalReceivedBytes)
			}
			sp.lock.RUnlock()
		}
	}
}

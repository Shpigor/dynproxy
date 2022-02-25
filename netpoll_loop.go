package dynproxy

import (
	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"
	"runtime"
)

type EventLoopConfig struct {
	Name            string
	LockOsThread    bool
	EventBufferSize int
}

type EventLoop struct {
	Name            string
	lockOsThread    bool
	eventBufferSize int
	isRunning       *atomic.Bool
	poller          Poller
	eventChan       chan Event
	sessionHolder   SessionHolder
}

func NewEventLoop(config EventLoopConfig) (*EventLoop, error) {
	if log.Debug().Enabled() {
		log.Debug().Msgf("init event loop:%+v", config)
	} else {
		log.Info().Msgf("init event loop:%s", config.Name)
	}

	poller, err := openPoller(config.EventBufferSize)
	if err != nil {
		log.Error().Msgf("can't open poller: %+v", err)
		return nil, err
	}
	eLoop := &EventLoop{
		Name:         config.Name,
		lockOsThread: config.LockOsThread,
		isRunning:    atomic.NewBool(false),
		poller:       poller,
	}
	return eLoop, nil
}

func (el *EventLoop) Start(handler NetEventHandler, holder SessionHolder) {
	if el.lockOsThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	el.isRunning.Store(true)
	for el.isRunning.Load() {
		_, err := el.poller.WaitForEvents(handler, holder)
		if err != nil {
			log.Error().Msgf("got error while waiting for the net events: %+v", err)
		}
	}
	defer el.poller.Close()
}

func (el *EventLoop) Stop() {
	el.isRunning.Store(false)
}

func (el *EventLoop) PollForRead(fd int) error {
	return el.poller.AddRead(fd)
}

func (el *EventLoop) DeletePoll(fd int) error {
	return el.poller.DeletePoll(fd)
}

func (el *EventLoop) PollForReadAndErrors(fds ...int) error {
	for _, fd := range fds {
		err := el.poller.AddReadErrors(fd)
		if err != nil {
			return err
		}
	}
	return nil
}

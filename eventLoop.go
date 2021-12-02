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
	poller          *Poller
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

func (el *EventLoop) Start(callback func(fd int, events uint32) error) {
	if el.lockOsThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	el.isRunning.Store(true)
	for el.isRunning.Load() {
		evCount, err := el.poller.waitForEvents(callback)
		if err != nil {
			log.Error().Msgf("got error while waiting for the net events: %+v", err)
		}
		if log.Debug().Enabled() {
			log.Debug().Msgf("processed %d netpoll events", evCount)
		}
	}
	defer el.poller.close()
}

func (el *EventLoop) Stop() {
	el.isRunning.Store(false)
}

func (el *EventLoop) PollForRead(fd int) error {
	return el.poller.addRead(fd)
}

func (el *EventLoop) DeletePoll(fd int) error {
	return el.poller.delete(fd)
}

func (el *EventLoop) PollForReadAndErrors(fd int) error {
	return el.poller.addReadErrors(fd)
}

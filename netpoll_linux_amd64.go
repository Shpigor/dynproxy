package dynproxy

import (
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
	"math"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

const (
	readEvents       = unix.EPOLLPRI | unix.EPOLLIN | unix.EPOLLET
	writeEvents      = unix.EPOLLOUT
	readWriteEvents  = readEvents | writeEvents
	errorEvents      = unix.EPOLLERR | unix.EPOLLHUP | unix.EPOLLRDHUP
	readErrorsEvents = readEvents | errorEvents
	allEvents        = readEvents | errorEvents | writeEvents
)

type LinuxPoller struct {
	eventBufferSize int
	eventQueueSize  int
	fd              int
	lockOSThread    bool
	events          []unix.EpollEvent
	timeout         int
}

func OpenPoller(eventsBufferSize int) (Poller, error) {
	fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, os.NewSyscallError("epoll_create1", err)
	}
	bufferSize := int(math.Max(float64(eventsBufferSize), defEventsBufferSize))
	return &LinuxPoller{
		eventBufferSize: bufferSize,
		fd:              fd,
		timeout:         blocked,
		events:          make([]unix.EpollEvent, bufferSize),
	}, nil
}

func (p *Poller) Close() {
	err := os.NewSyscallError("close", unix.Close(p.fd))
	if err != nil {
		log.Error().Msgf("got error while closing epoll: %+v", err)
	}
}

func (p *Poller) WaitForEvents(handler NetEventHandler, provider SessionHolder) (int, error) {
	evCount, err := epollWait(p.fd, p.events, p.timeout)
	if evCount == 0 || (evCount < 0 && err == unix.EINTR) {
		runtime.Gosched()
	} else if err != nil {
		log.Printf("error occurs in epoll: %v", os.NewSyscallError("epoll_wait", err))
		return 0, err
	}
	for i := 0; i < evCount; i++ {
		event := p.events[i]
		fd := int(event.Fd)
		log.Debug().Msgf("[%d] epoll event:%d", fd, event)
		session, err := provider.FindSessionByFd(fd)
		if err != nil {
			err := p.deletePoll(fd)
			if err != nil {
				log.Error().Msgf("[%d] error occurs while detaching fd from netpoll: %v", fd, err)
			}
			continue
		}
		if readEvents&event.Events > 0 {
			err = handler.ReadEvent(session, fd)
		}
		if errorEvents&event.Events > 0 {
			err = handler.ErrorEvent(session, parseErrors(event.Events))
		}
		if err != nil {
			fds := session.GetFds()
			for _, fd := range fds {
				err := p.deletePoll(fd)
				if err != nil {
					log.Error().Msgf("[%d] error occurs while detaching fd from netpoll: %v", fd, err)
				}
			}
			if err != closedSession {
				log.Error().Msgf("[%d] error occurs in event-loop: %v", fd, err)
				err := session.Close()
				if err != nil {
					log.Error().Msgf("[%d] error occurs while closing session: %v", fd, err)
				}
			}
			provider.RemoveSession(session)
		}
	}
	return evCount, nil
}

func parseErrors(events uint32) []error {
	if errorEvents&events > 0 {

	}
	return nil
}

func (p *Poller) AddReadErrors(fd int) error {
	if log.Debug().Enabled() {
		log.Debug().Msgf("add read|errors epoll for fd: %d", fd)
	}
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: readErrorsEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl add", err)
	}
	return nil
}

func (p *Poller) AddReadWrite(fd int) error {
	if log.Debug().Enabled() {
		log.Debug().Msgf("add read|write epoll for fd: %d", fd)
	}
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: readWriteEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl add", err)
	}
	return nil
}

func (p *Poller) AddRead(fd int) error {
	if log.Debug().Enabled() {
		log.Debug().Msgf("add read epoll for fd: %d", fd)
	}
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: readEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl add", err)
	}
	return nil
}

func (p *Poller) AddWrite(fd int) error {
	if log.Debug().Enabled() {
		log.Debug().Msgf("add write epoll for fd: %d", fd)
	}
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: writeEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl add", err)
	}
	return nil
}

func (p *Poller) DeletePoll(fd int) error {
	if log.Debug().Enabled() {
		log.Debug().Msgf("delete epoll for fd: %d", fd)
	}
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return os.NewSyscallError("epoll_ctl del", err)
	}
	return nil
}

func (p *Poller) AddError(fd int) error {
	if log.Debug().Enabled() {
		log.Debug().Msgf("add error epoll for fd: %d", fd)
	}
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: errorEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl del", err)
	}
	return nil
}

func epollWait(epollFd int, events []unix.EpollEvent, msec int) (count int, err error) {
	var eventCount uintptr
	var eventsPointer = unsafe.Pointer(&events[0])
	if msec == 0 {
		eventCount, _, err = syscall.RawSyscall6(syscall.SYS_EPOLL_PWAIT, uintptr(epollFd), uintptr(eventsPointer), uintptr(len(events)), 0, 0, 0)
	} else {
		eventCount, _, err = syscall.Syscall6(syscall.SYS_EPOLL_PWAIT, uintptr(epollFd), uintptr(eventsPointer), uintptr(len(events)), uintptr(msec), 0, 0)
	}
	if err == syscall.Errno(0) {
		err = nil
	}
	return int(eventCount), err
}

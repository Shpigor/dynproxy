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
	readEvents       = unix.EPOLLPRI | unix.EPOLLIN
	writeEvents      = unix.EPOLLOUT
	readWriteEvents  = readEvents | writeEvents
	errorEvents      = unix.EPOLLERR | unix.EPOLLHUP
	readErrorsEvents = readEvents | errorEvents
)

func openPoller(eventsBufferSize int) (*Poller, error) {
	fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, os.NewSyscallError("epoll_create1", err)
	}
	bufferSize := int(math.Max(float64(eventsBufferSize), defEventsBufferSize))
	return &Poller{
		eventBufferSize: bufferSize,
		fd:              fd,
		timeout:         blocked,
		events:          make([]unix.EpollEvent, bufferSize),
	}, nil
}

func (p *Poller) close() {
	err := os.NewSyscallError("close", unix.Close(p.fd))
	if err != nil {
		log.Error().Msgf("got error while closing epoll: %+v", err)
	}
}

func (p *Poller) waitForEvents(callback func(fd int, events uint32) error) (int, error) {
	//evCount, err := unix.EpollWait(p.fd, p.events, p.timeout)
	evCount, err := epollWait(p.fd, p.events, p.timeout)
	if evCount == 0 || (evCount < 0 && err == unix.EINTR) {
		p.timeout = blocked
		runtime.Gosched()
	} else if err != nil {
		log.Printf("error occurs in epoll: %v", os.NewSyscallError("epoll_wait", err))
		return 0, err
	}
	for i := 0; i < evCount; i++ {
		event := p.events[i]
		fd := int(event.Fd)
		err = callback(fd, event.Events)
		if err != nil {
			log.Error().Msgf("error occurs in event-loop: %v", err)
		}
	}
	return evCount, nil
}

func (p *Poller) addReadErrors(fd int) error {
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: readErrorsEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl add", err)
	}
	return nil
}

func (p *Poller) addReadWrite(fd int) error {
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: readWriteEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl add", err)
	}
	return nil
}

func (p *Poller) addRead(fd int) error {
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: readEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl add", err)
	}
	return nil
}

func (p *Poller) addWrite(fd int) error {
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: writeEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl add", err)
	}
	return nil
}

func (p *Poller) delete(fd int) error {
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return os.NewSyscallError("epoll_ctl del", err)
	}
	return nil
}

func (p *Poller) addError(fd int) error {
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: errorEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl del", err)
	}
	return nil
}

func epollWait(epfd int, events []unix.EpollEvent, msec int) (n int, err error) {
	var r0 uintptr
	var _p0 = unsafe.Pointer(&events[0])
	if msec == 0 {
		r0, _, err = syscall.RawSyscall6(syscall.SYS_EPOLL_PWAIT, uintptr(epfd), uintptr(_p0), uintptr(len(events)), 0, 0, 0)
	} else {
		r0, _, err = syscall.Syscall6(syscall.SYS_EPOLL_PWAIT, uintptr(epfd), uintptr(_p0), uintptr(len(events)), uintptr(msec), 0, 0)
	}
	if err == syscall.Errno(0) {
		err = nil
	}
	return int(r0), err
}

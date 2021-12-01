package dynproxy

import (
	"github.com/panjf2000/gnet/errors"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"runtime"
)

const eventsChanSize = 1024
const eventsBufferSize = 32

const (
	readEvents       = unix.EPOLLPRI | unix.EPOLLIN
	writeEvents      = unix.EPOLLOUT
	readWriteEvents  = readEvents | writeEvents
	errorEvents      = unix.EPOLLERR | unix.EPOLLHUP
	readErrorsEvents = readEvents | errorEvents
)

type Poller struct {
	fd           int // epoll fd
	lockOSThread bool
	events       chan *unix.EpollEvent
}

func OpenPoller() (*Poller, error) {
	fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, os.NewSyscallError("epoll_create1", err)
	}

	return &Poller{
		fd:     fd,
		events: make(chan *unix.EpollEvent, eventsChanSize),
	}, nil
}

func (p *Poller) Close() error {
	err := os.NewSyscallError("close", unix.Close(p.fd))
	if err != nil {
		return err
	}
	return nil
}

func (p *Poller) Polling(callback func(fd int, ev uint32) error) error {
	events := make([]unix.EpollEvent, eventsBufferSize)

	msec := -1
	for {
		n, err := unix.EpollWait(p.fd, events, msec)
		if n == 0 || (n < 0 && err == unix.EINTR) {
			msec = -1
			runtime.Gosched()
			continue
		} else if err != nil {
			log.Printf("error occurs in epoll: %v", os.NewSyscallError("epoll_wait", err))
			return err
		}
		msec = 0

		for i := 0; i < n; i++ {
			ev := events[i]
			fd := int(ev.Fd)
			err = callback(fd, ev.Events)
			if err != nil {
				if err == errors.ErrAcceptSocket || err == errors.ErrServerShutdown {
					return err
				}
				log.Printf("error occurs in event-loop: %v", err)
			}
		}
	}
}

type PollAttachment struct {
	FD int
}

func (p *Poller) AddReadErrors(fd int) error {
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: readErrorsEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl add", err)
	}
	return nil
}

func (p *Poller) AddReadWrite(fd int) error {
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: readWriteEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl add", err)
	}
	return nil
}

func (p *Poller) AddRead(fd int) error {
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: readEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl add", err)
	}
	return nil
}

func (p *Poller) AddWrite(fd int) error {
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: writeEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl add", err)
	}
	return nil
}

func (p *Poller) ModRead(fd int) error {
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_MOD, fd, &unix.EpollEvent{Fd: int32(fd), Events: readEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl mod", err)
	}
	return nil
}

func (p *Poller) ModReadWrite(fd int) error {
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_MOD, fd, &unix.EpollEvent{Fd: int32(fd), Events: readWriteEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl mod", err)
	}
	return nil
}

func (p *Poller) Delete(fd int) error {
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return os.NewSyscallError("epoll_ctl del", err)
	}
	return nil
}

func (p *Poller) AddError(fd int) error {
	err := unix.EpollCtl(p.fd, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: errorEvents})
	if err != nil {
		return os.NewSyscallError("epoll_ctl del", err)
	}
	return nil
}

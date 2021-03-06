package dynproxy

import (
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
	"net"
	"syscall"
)

func setSocketOptions(conn net.Conn) int {
	fd, connType, err := ConnToFileDesc(conn)
	if err != nil {
		log.Error().Msgf("error occur while getting file descriptor from connection:%+v", err)
		return -1
	}
	switch connType {
	case TCP:
		setTcpSocketOptions(fd)
	case TLS:
		setTlsSocketOptions(fd)
	case UNKNOWN:
		log.Error().Msg("error occur while setting socket options for unknown connection type")
	}
	return fd
}

func setTcpSocketOptions(fd int) {
	err := unix.SetNonblock(fd, true)
	if err != nil {
		log.Error().Msgf("got error while setting socket options O_NONBLOCK: %+v", err)
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF, 8192)
	if err != nil {
		log.Error().Msgf("got error while setting socket options SO_RCVBUF: %+v", err)
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_SNDBUF, 8192)
	if err != nil {
		log.Error().Msgf("got error while setting socket options SO_SNDBUF: %+v", err)
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		log.Error().Msgf("got error while setting socket options SO_REUSEADDR: %+v", err)
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	if err != nil {
		log.Error().Msgf("got error while setting socket options SO_REUSEPORT: %+v", err)
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1)
	if err != nil {
		log.Error().Msgf("got error while setting socket options SO_KEEPALIVE: %+v", err)
	}
	err = syscall.SetsockoptLinger(fd, syscall.SOL_SOCKET, syscall.SO_LINGER, &syscall.Linger{Onoff: 0, Linger: 0})
	if err != nil {
		log.Error().Msgf("got error while setting socket options SO_LINGER: %+v", err)
	}
}

func setTlsSocketOptions(fd int) {
	err := unix.SetNonblock(fd, true)
	if err != nil {
		log.Error().Msgf("got error while setting socket options O_NONBLOCK: %+v", err)
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF, 8192)
	if err != nil {
		log.Error().Msgf("got error while setting socket options SO_RCVBUF: %+v", err)
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_SNDBUF, 8192)
	if err != nil {
		log.Error().Msgf("got error while setting socket options SO_SNDBUF: %+v", err)
	}
	err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_INCOMING_CPU, 0)
	if err != nil {
		log.Error().Msgf("got error while setting socket options SO_INCOMING_CPU: %+v", err)
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		log.Error().Msgf("got error while setting socket options SO_REUSEADDR: %+v", err)
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	if err != nil {
		log.Error().Msgf("got error while setting socket options SO_REUSEPORT: %+v", err)
	}
	err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1)
	if err != nil {
		log.Error().Msgf("got error while setting socket options SO_KEEPALIVE: %+v", err)
	}
	err = syscall.SetsockoptLinger(fd, syscall.SOL_SOCKET, syscall.SO_LINGER, &syscall.Linger{Onoff: 0, Linger: 0})
	if err != nil {
		log.Error().Msgf("got error while setting socket options SO_LINGER: %+v", err)
	}
}

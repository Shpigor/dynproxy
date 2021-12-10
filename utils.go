package dynproxy

import (
	"crypto/tls"
	"errors"
	"net"
	"os"
	"reflect"
	"unsafe"
)

const (
	UNKNOWN ConnType = iota
	TCP
	TLS
	UDP
)

type FileDesc interface {
	File() (f *os.File, err error)
}

type ConnType int

func tlsConnToFileDesc(tlsConn *tls.Conn) FileDesc {
	// XXX: This is really BAD!!! Only way currently to get the underlying
	// connection of the tls.Conn. At least until
	// https://github.com/golang/go/issues/29257 is solved.
	conn := reflect.ValueOf(tlsConn).Elem().FieldByName("conn")
	conn = reflect.NewAt(conn.Type(), unsafe.Pointer(conn.UnsafeAddr())).Elem()
	return conn.Interface().(FileDesc)
}

func ConnToFileDesc(conn net.Conn) (int, ConnType, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if ok {
		file, err := tcpConn.File()
		if err != nil {
			return 0, TCP, err
		}
		return int(file.Fd()), TCP, nil
	} else {
		tls, ok := conn.(*tls.Conn)
		if ok {
			conn := reflect.ValueOf(tls).Elem().FieldByName("conn")
			conn = reflect.NewAt(conn.Type(), unsafe.Pointer(conn.UnsafeAddr())).Elem()
			fileDesc := conn.Interface().(FileDesc)
			file, err := fileDesc.File()
			if err != nil {
				return 0, TLS, err
			}
			return int(file.Fd()), TLS, nil
		}
	}
	return 0, UNKNOWN, errors.New("can't cast net.Conn to *net.TCPConn")
}

package dynproxy

import (
	"crypto/tls"
	"errors"
	"net"
	"os"
	"reflect"
	"unsafe"
)

type FileDesc interface {
	File() (f *os.File, err error)
}

func tlsConnToFileDesc(tlsConn *tls.Conn) FileDesc {
	// XXX: This is really BAD!!! Only way currently to get the underlying
	// connection of the tls.Conn. At least until
	// https://github.com/golang/go/issues/29257 is solved.
	conn := reflect.ValueOf(tlsConn).Elem().FieldByName("conn")
	conn = reflect.NewAt(conn.Type(), unsafe.Pointer(conn.UnsafeAddr())).Elem()
	return conn.Interface().(FileDesc)
}

func ConnToFileDesc(conn net.Conn) (uintptr, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if ok {
		file, err := tcpConn.File()
		if err != nil {
			return 0, err
		}
		return file.Fd(), nil
	} else {
		tls, ok := conn.(*tls.Conn)
		if ok {
			conn := reflect.ValueOf(tls).Elem().FieldByName("conn")
			conn = reflect.NewAt(conn.Type(), unsafe.Pointer(conn.UnsafeAddr())).Elem()
			fileDesc := conn.Interface().(FileDesc)
			file, err := fileDesc.File()
			if err != nil {
				return 0, err
			}
			return file.Fd(), nil
		}
	}
	return 0, errors.New("can't cast net.Conn to *net.TCPConn")
}

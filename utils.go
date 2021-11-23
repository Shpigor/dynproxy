package dynproxy

import (
	"crypto/tls"
	"net"
	"os"
	"reflect"
	"unsafe"
)

type FileDesc interface {
	File() (f *os.File, err error)
}

func TLSConnToNetConn(tlsConn *tls.Conn) net.Conn {
	// XXX: This is really BAD!!! Only way currently to get the underlying
	// connection of the tls.Conn. At least until
	// https://github.com/golang/go/issues/29257 is solved.
	conn := reflect.ValueOf(tlsConn).Elem().FieldByName("conn")
	conn = reflect.NewAt(conn.Type(), unsafe.Pointer(conn.UnsafeAddr())).Elem()
	return conn.Interface().(net.Conn)
}

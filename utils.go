package dynproxy

import (
	"crypto/tls"
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

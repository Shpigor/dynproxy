package dynproxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net"
	"syscall"
)

type Frontend struct {
	Context         context.Context
	Net             string
	Address         string
	Name            string
	defaultBalancer string
	TlsConfig       *TlsConfig
	connChannel     chan *newConn
	ocspProc        *OCSPProcessor
}

type TlsConfig struct {
	SkipVerify bool
	CACertPath string
	CertPath   string
	PkPath     string
	// Init phase
	Certificates map[uint16]*tls.Certificate
	caCertPool   *x509.CertPool
}

func (f *Frontend) Listen() error {
	listener, err := f.listen()
	if err != nil {
		return err
	}
	if f.TlsConfig != nil {
		listener = f.listenTls(listener)
		go f.handleTlsAccept(listener)
	} else {
		go f.handleAccept(listener)
	}
	return nil
}
func (f *Frontend) handleAccept(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("got error while accept connection: %+v", err)
		}
		tcpConn, ok := conn.(*net.TCPConn)
		if !ok {
			fmt.Println("error in casting *net.Conn to *net.TCPConn!")
		} else {
			configureSocket(tcpConn)
			f.handleNewConnection(tcpConn)
		}
	}
}

func (f *Frontend) handleTlsAccept(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("got error while accept connection: %+v", err)
		}
		tlsConn, ok := conn.(*tls.Conn)
		if !ok {
			fmt.Println("error in casting *net.Conn to *net.TCPConn!")
		} else {
			err := tlsConn.Handshake()
			if err != nil {
				log.Printf("TLS handshake error: %+v", err)
				//log.Printf("TLS handshake status:[%+v]", tlsConn)
				tlsConn.Close()
				// TODO: notify about client error
				continue
			}
			configureSocket(tlsConnToFileDesc(tlsConn))
			f.handleNewConnection(tlsConn)
		}
	}
}

func configureSocket(fd FileDesc) {
	file, err := fd.File()
	if err != nil {
		log.Error().Msgf("error in getting file for the connection:%+v", err)
	}
	f := int(file.Fd())
	err = syscall.SetsockoptInt(f, syscall.SOL_SOCKET, syscall.SO_RCVBUF, 8192)
	if err != nil {
		log.Error().Msgf("got error while setting socket options SO_RCVBUF: %+v", err)
	}
	err = syscall.SetsockoptInt(f, syscall.SOL_SOCKET, syscall.SO_SNDBUF, 8192)
	if err != nil {
		log.Error().Msgf("got error while setting socket options SO_SNDBUF: %+v", err)
	}
}

func (f *Frontend) listenTls(listener net.Listener) net.Listener {
	f.initTlsConfig()
	config := &tls.Config{
		InsecureSkipVerify:    f.TlsConfig.SkipVerify,
		ClientAuth:            tls.RequireAndVerifyClientCert,
		ClientCAs:             f.TlsConfig.caCertPool,
		RootCAs:               f.TlsConfig.caCertPool,
		GetCertificate:        f.getFrontendCert,
		VerifyPeerCertificate: f.verifyClientCert,
		//ClientSessionCache: tls.NewLRUClientSessionCache(500),
	}
	return tls.NewListener(listener, config)
}

func (f *Frontend) listen() (net.Listener, error) {
	return net.Listen(f.Net, f.Address)
}

func (f *Frontend) getFrontendCert(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	//cipherSuites := info.CipherSuites
	certificate := f.TlsConfig.Certificates[0]
	if certificate != nil {
		return certificate, nil
	}
	return nil, errors.New("can't find valid certificate")
}

func (f *Frontend) verifyClientCert(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	_, err := x509.ParseCertificate(rawCerts[0])
	return err
}

func (f *Frontend) initTlsConfig() {
	caCertPool := x509.NewCertPool()
	f.TlsConfig.caCertPool = caCertPool
	caCert, err := parseCaCertFile(f.TlsConfig.CACertPath)
	if err != nil {
		log.Fatal().Msgf("got error while loading CA cert: %+v", err)
	}
	caCertPool.AddCert(caCert)

	cert, err := parseCertFile(f.TlsConfig.CertPath, f.TlsConfig.PkPath)
	if err != nil {
		log.Fatal().Msgf("got error while loading frontend certificate: %+v", err)
	}
	err = f.addOcspStaple(&cert, caCert)
	if err != nil {
		log.Fatal().Msgf("got error while verify(ocsp) frontend certificate: %+v", err)
	}

	f.TlsConfig.Certificates = make(map[uint16]*tls.Certificate)
	f.TlsConfig.Certificates[0] = &cert
}

func (f *Frontend) addOcspStaple(cert *tls.Certificate, caCert *x509.Certificate) error {
	if f.ocspProc != nil && f.ocspProc.ocspStapleEnabled {
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return err
		}
		ocspStaple, err := f.ocspProc.OcspVerify(x509Cert, caCert)
		if err != nil {
			return err
		}
		cert.OCSPStaple = ocspStaple
	}
	return nil
}

func (f *Frontend) handleNewConnection(conn net.Conn) {
	f.connChannel <- &newConn{
		frontend: conn,
		backend:  f.defaultBalancer,
	}
}

func parseCaCertFile(filename string) (*x509.Certificate, error) {
	ct, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(ct)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func parseCertFile(certFile, pkFile string) (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certFile, pkFile)
}

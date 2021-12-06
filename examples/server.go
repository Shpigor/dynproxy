package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"dynproxy"
	"encoding/pem"
	"flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net"
	"sync"
)

var serverIp string
var serverPort int
var privateKeyPath string
var eventLoop *dynproxy.EventLoop

func init() {
	flag.StringVar(&serverIp, "ip", "10.0.0.81", "listening ip address.")
	flag.StringVar(&privateKeyPath, "pk", "/home/igor/ca/server.pk", "path to private key")
	flag.IntVar(&serverPort, "p", 3030, "listening port.")
	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

type server struct {
	pk      *rsa.PrivateKey
	address *net.TCPAddr
	lock    *sync.RWMutex
	wg      *sync.WaitGroup
	bb      []byte
	streams map[int]*dynproxy.Stream
}

func (s *server) ReadEvent(stream *dynproxy.Stream, direction int) error {
	conn := stream.GetConnByDirection(direction)
	read, err := conn.Read(s.bb)
	if err != nil {
		if err.Error() == "EOF" {
			stream.Close(direction)
			s.RemoveStream(stream)
		} else {
			log.Error().Msgf("got error while reading data from connection %+v", err)
			return err
		}
	}
	if read > 0 {
		msg := s.bb[:read]
		log.Info().Msgf(">> %s", string(msg))
		message, err := prepareResponseSignature(msg, s.pk)
		if err != nil {
			log.Error().Msgf("got error while preparing signed response %+v", err)
		}
		_, err = conn.Write(message)
		if err != nil {
			log.Error().Msgf("got error while writing signed response %+v", err)
		}
	}
	return nil
}

func (s *server) WriteEvent(stream *dynproxy.Stream, direction int) error {
	return nil
}

func (s *server) ErrorEvent(stream *dynproxy.Stream, direction int, errors []error) error {
	err := stream.Close(direction)
	if err != nil {
		log.Error().Msgf("got error while remove epoll: %+v", err)
		return err
	}
	return nil
}

func (s *server) RemoveByFd(fd int) {
	s.lock.Lock()
	delete(s.streams, fd)
	s.lock.Unlock()
}

func (s *server) RemoveStream(stream *dynproxy.Stream) {
	s.lock.Lock()
	delete(s.streams, stream.GetFd())
	s.lock.Unlock()
}

func (s *server) FindStreamByFd(fd int) (*dynproxy.Stream, int) {
	s.lock.RLock()
	stream := s.streams[fd]
	s.lock.RUnlock()
	return stream, 0
}

func (s *server) AddStream(fd int, stream *dynproxy.Stream) {
	s.lock.Lock()
	s.streams[fd] = stream
	s.lock.Unlock()
}

func main() {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	address := &net.TCPAddr{
		IP:   net.ParseIP(serverIp),
		Port: serverPort,
	}
	pk, err := parsePk(privateKeyPath)
	if err != nil {
		log.Fatal().Msgf("can't read private key: %+v", err)
	}
	srv := server{
		address: address,
		pk:      pk,
		lock:    &sync.RWMutex{},
		wg:      wg,
		bb:      make([]byte, 2048),
		streams: make(map[int]*dynproxy.Stream),
	}
	eventLoop, err = dynproxy.NewEventLoop(dynproxy.EventLoopConfig{
		Name:            "MainLoop",
		EventBufferSize: 256,
		LockOsThread:    true,
	})
	if err != nil {
		log.Fatal().Msgf("can't create event loop: %+v", err)
	}

	go eventLoop.Start(&srv, &srv)
	srv.listenTcp()
	wg.Wait()
}

func (s *server) listenTcp() {
	tcp, err := net.ListenTCP("tcp", s.address)
	if err != nil {
		s.wg.Done()
		log.Fatal().Msgf("got error while listening socket: %+v", err)
	}
	go s.handleAcceptConnection(tcp)
}

func parsePk(path string) (*rsa.PrivateKey, error) {
	pk, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pkPemBlock, _ := pem.Decode(pk)
	pkPemBytes := pkPemBlock.Bytes
	var parsedKey *rsa.PrivateKey
	if parsedKey, err = x509.ParsePKCS1PrivateKey(pkPemBytes); err != nil {
		pkInterface, err := x509.ParsePKCS8PrivateKey(pkPemBytes)
		if err != nil {
			return nil, err
		}
		parsedKey = pkInterface.(*rsa.PrivateKey)
	}
	return parsedKey, nil
}

func (s *server) handleAcceptConnection(ln *net.TCPListener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Info().Msgf("got error while eccepting tcp session: %+v", err)
			break
		}
		fdConn, ok := conn.(dynproxy.FileDesc)
		if !ok {
			log.Info().Msgf("can't cast net.Conn to FileDesc %+v", conn)
		}
		file, err := fdConn.File()
		if err != nil {
			log.Info().Msgf("can't get file from FileDesc %+v", err)
		}

		fd := int(file.Fd())
		s.AddStream(fd, dynproxy.NewStream(fd, conn))
		err = eventLoop.PollForReadAndErrors(fd)
		if err != nil {
			log.Info().Msgf("can't add read fd to epoll %+v", err)
		}
	}
	log.Info().Msg("finished to accepting tcp connections")
}

func prepareResponseSignature(message []byte, pk *rsa.PrivateKey) ([]byte, error) {
	msgHash := sha256.New()
	_, err := msgHash.Write(message)
	if err != nil {
		return nil, err
	}
	msgHashSum := msgHash.Sum(nil)
	return rsa.SignPSS(rand.Reader, pk, crypto.SHA256, msgHashSum, nil)
}

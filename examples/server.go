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
	"golang.org/x/sys/unix"
	"io/ioutil"
	"net"
	"sync"
)

var serverIp string
var serverPort int
var privateKeyPath string
var connections map[int]net.Conn

var lock = sync.RWMutex{}
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

			lock.Lock()
			delete(connections, fd)
			lock.Unlock()
		} else {
			log.Error().Msgf("got error while reading data from connection %+v", err)
			return err
		}
	}
	if read > 0 {
		msg := string(s.bb[:read])
		log.Info().Msgf(">> %s", msg)
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

func (s *server) CloseEvent(stream *dynproxy.Stream, direction int) error {
	err := stream.Close(direction)
	if err != nil {
		log.Error().Msgf("got error while remove epoll: %+v", err)
		return err
	}
	return nil
}

func (s *server) FindStreamByFd(fd int) (*dynproxy.Stream, int) {
	lock.RLock()
	stream := s.streams[fd]
	lock.RUnlock()
	return stream, 0
}

func (s *server) AddStream(fd int, stream *dynproxy.Stream) {
	lock.Lock()
	s.streams[fd] = stream
	lock.Unlock()
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
	go handleAcceptConnection(tcp)
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

func handleAcceptConnection(ln *net.TCPListener) {
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
		saveConnWithFd(fd, conn)
		err = eventLoop.PollForReadAndErrors(fd)
		if err != nil {
			log.Info().Msgf("can't add read fd to epoll %+v", err)
		}
	}
	log.Info().Msg("finished to accepting tcp connections")
}

func epollReadCallback(pk *rsa.PrivateKey) func(fd int, ev uint32) error {
	bb := make([]byte, 2048)
	return func(fd int, ev uint32) error {
		conn, ok := getConnByFd(fd)
		if ok {
			switch ev {
			case unix.EPOLLIN:
				fallthrough
			case unix.EPOLLPRI:
				fallthrough
			case unix.EPOLLOUT:
				err := handleRequest(fd, conn, bb, pk)
				if err != nil {
					return err
				}
			case unix.EPOLLHUP:
				log.Error().Msgf("connection fd:%d is susspended\n", fd)
				removeConnByFd(fd)
			case unix.EPOLLERR:
				log.Error().Msgf("got error for connection fd:%d\n", fd)
				removeConnByFd(fd)
			}
		} else {
			removeConnByFd(fd)
		}
		return nil
	}
}

func handleRequest(fd int, conn net.Conn, bb []byte, pk *rsa.PrivateKey) error {
	read, err := conn.Read(bb)
	if err != nil {
		if err.Error() == "EOF" {
			conn.Close()
			lock.Lock()
			delete(connections, fd)
			lock.Unlock()
		} else {
			log.Error().Msgf("got error while reading data from connection %+v", err)
			return err
		}
	}
	if read > 0 {
		msg := string(bb[:read])
		log.Info().Msgf(">> %s", msg)
		message, err := prepareResponseSignature(msg, pk)
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

func prepareResponseSignature(message string, pk *rsa.PrivateKey) ([]byte, error) {
	msg := []byte(message)

	msgHash := sha256.New()
	_, err := msgHash.Write(msg)
	if err != nil {
		return nil, err
	}
	msgHashSum := msgHash.Sum(nil)

	return rsa.SignPSS(rand.Reader, pk, crypto.SHA256, msgHashSum, nil)
}

func getConnByFd(fd int) (net.Conn, bool) {
	lock.RLock()
	conn, ok := connections[fd]
	lock.RUnlock()
	return conn, ok
}

func removeConnByFd(fd int) {
	lock.RLock()
	delete(connections, fd)
	lock.RUnlock()
	err := eventLoop.DeletePoll(fd)
	if err != nil {
		log.Error().Msgf("got error while remove epoll: %+v", err)
	}
}

func saveConnWithFd(fd int, conn net.Conn) {
	lock.Lock()
	connections[fd] = conn
	lock.Unlock()
}

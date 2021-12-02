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

func main() {
	group := &sync.WaitGroup{}
	group.Add(1)
	address := &net.TCPAddr{
		IP:   net.ParseIP(serverIp),
		Port: serverPort,
	}
	pk, err := parsePk(privateKeyPath)
	if err != nil {
		log.Fatal().Msgf("can't read private key: %+v", err)
	}
	eventLoop, err = dynproxy.NewEventLoop(dynproxy.EventLoopConfig{
		Name:            "MainLoop",
		EventBufferSize: 256,
		LockOsThread:    true,
	})
	if err != nil {
		log.Fatal().Msgf("can't create event loop: %+v", err)
	}
	connections = make(map[int]net.Conn)
	go eventLoop.Start(epollReadCallback(pk))
	tcpServer(group, pk, address)
	group.Wait()
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

func tcpServer(group *sync.WaitGroup, pk *rsa.PrivateKey, address *net.TCPAddr) {
	tcp, err := net.ListenTCP("tcp", address)
	if err != nil {
		group.Done()
		log.Fatal().Msgf("got error while listening socket: %+v", err)
	}
	go handleEpollTcpSocketEvents(tcp)
}

func handleEpollTcpSocketEvents(ln *net.TCPListener) {
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
		lock.Lock()
		connections[fd] = conn
		lock.Unlock()
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
		lock.RLock()
		conn, ok := connections[fd]
		lock.RUnlock()
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
				delete(connections, fd)
			case unix.EPOLLERR:
				log.Error().Msgf("got error for connection fd:%d\n", fd)
				delete(connections, fd)
			}
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

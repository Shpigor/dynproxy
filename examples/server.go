package main

import (
	"context"
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
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

type server struct {
	pk        *rsa.PrivateKey
	address   *net.TCPAddr
	wg        *sync.WaitGroup
	streams   dynproxy.SessionHolder
	handler   dynproxy.NetEventHandler
	eventChan chan dynproxy.Event
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
		address:   address,
		pk:        pk,
		wg:        wg,
		streams:   dynproxy.NewMapSessionProvider(context.Background()),
		handler:   dynproxy.NewBufferHandler(),
		eventChan: make(chan dynproxy.Event, 100),
	}
	eventLoop, err = dynproxy.NewEventLoop(dynproxy.EventLoopConfig{
		Name:            "MainLoop",
		EventBufferSize: 256,
		LockOsThread:    true,
	})
	if err != nil {
		log.Fatal().Msgf("can't create event loop: %+v", err)
	}

	go eventLoop.Start(srv.handler, srv.streams)
	srv.listenTcp()
	wg.Wait()
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

func (s *server) listenTcp() {
	tcp, err := net.ListenTCP("tcp", s.address)
	if err != nil {
		s.wg.Done()
		log.Fatal().Msgf("got error while listening socket: %+v", err)
	}
	go s.handleAcceptConnection(tcp)
}

func (s *server) handleAcceptConnection(ln *net.TCPListener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Info().Msgf("got error while eccepting tcp session: %+v", err)
			break
		}
		session, err := dynproxy.NewClientSession(conn, s.eventChan, s.handleRead)
		if err != nil {
			log.Error().Msgf("can't create new client session %+v", err)
			continue
		}
		s.streams.AddSession(session)
		err = eventLoop.PollForReadAndErrors(session.GetFds()...)
		if err != nil {
			log.Info().Msgf("can't add read fd to epoll %+v", err)
		}
	}
	log.Info().Msg("finished to accepting tcp connections")
}

func (s *server) handleRead(src, dst net.Conn, buffer []byte) error {
	read, err := src.Read(buffer)
	if err != nil {
		log.Error().Msgf("got error while reading data from connection %+v", err)
		return err
	}
	if read > 0 {
		msg := buffer[:read]
		log.Info().Msgf(">> %s", string(msg))
		message, err := prepareResponseSignature(msg, s.pk)
		if err != nil {
			log.Error().Msgf("got error while preparing signed response %+v", err)
		}
		_, err = dst.Write(message)
		if err != nil {
			log.Error().Msgf("got error while writing signed response %+v", err)
		}
	}
	return nil
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

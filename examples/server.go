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
	"golang.org/x/sys/unix"
	"io/ioutil"
	"log"
	"net"
	"sync"
)

var serverIp string
var serverPort int
var privateKeyPath string
var connections map[int]net.Conn

var lock = sync.RWMutex{}
var poller *dynproxy.Poller

func init() {
	flag.StringVar(&serverIp, "ip", "10.0.0.81", "listening ip address.")
	flag.StringVar(&privateKeyPath, "pk", "/home/igor/ca/server.pk", "path to private key")
	flag.IntVar(&serverPort, "p", 3030, "listening port.")
	flag.Parse()
}

func main() {
	address := &net.TCPAddr{
		IP:   net.ParseIP(serverIp),
		Port: serverPort,
	}
	connections = make(map[int]net.Conn)
	var err error
	poller, err = dynproxy.OpenPoller()
	if err != nil {
		log.Fatalf("got error while openning poller %+v", err)
	}
	group := &sync.WaitGroup{}
	group.Add(1)
	pk, err := parsePk(privateKeyPath)
	if err != nil {
		log.Fatalf("can't read private key: %+v", err)
	}
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
		log.Fatalf("got error while listening socket: %+v", err)
	}
	//go handleTcpSession(tcp, pk)
	go handleEpollTcpSocketEvents(tcp, pk)
}

func handleTcpSession(ln *net.TCPListener, pk *rsa.PrivateKey) {
	for {
		accept, err := ln.Accept()
		if err != nil {
			log.Printf("got error while eccepting tcp session: %+v", err)
			break
		}
		go readWriteConn(accept, pk)
	}
	log.Println("finished to accepting tcp connections")
}

func readWriteConn(conn net.Conn, pk *rsa.PrivateKey) {
	bb := make([]byte, 2048)
	defer conn.Close()
	for {
		read, err := conn.Read(bb)
		if err != nil {
			if err.Error() != "EOF" {
				log.Printf("got error while reading data from connection %+v", err)
			}
			break
		}
		if read > 0 {
			msg := string(bb[:read])
			log.Println(">> " + msg)
			message, err := prepareResponseSignature(msg, pk)
			if err != nil {
				log.Printf("got error while preparing signed response %+v", err)
			}
			_, err = conn.Write(message)
			if err != nil {
				log.Printf("got error while writing signed response %+v", err)
				break
			}
		}
	}
}

func handleEpollTcpSocketEvents(ln *net.TCPListener, pk *rsa.PrivateKey) {
	go func() {
		err := poller.Polling(epollReadCallback(pk))
		if err != nil {
			if err != nil {
				log.Printf("got error while polling events %+v", err)
			}
		}
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("got error while eccepting tcp session: %+v", err)
			break
		}
		fdConn, ok := conn.(dynproxy.FileDesc)
		if !ok {
			log.Printf("can't cast net.Conn to FileDesc %+v", conn)
		}
		file, err := fdConn.File()
		if err != nil {
			log.Printf("can't get file from FileDesc %+v", err)
		}

		fd := int(file.Fd())
		err = poller.AddReadErrors(fd)
		if err != nil {
			log.Printf("can't add read fd to epoll %+v", err)
		}
		lock.Lock()
		connections[fd] = conn
		lock.Unlock()
	}
	log.Println("finished to accepting tcp connections")
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
				log.Printf("connection fd:%d is susspended\n", fd)
				delete(connections, fd)
			case unix.EPOLLERR:
				log.Printf("got error for connection fd:%d\n", fd)
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
			log.Printf("got error while reading data from connection %+v", err)
			return err
		}
	}
	if read > 0 {
		msg := string(bb[:read])
		log.Println(">> " + msg)
		message, err := prepareResponseSignature(msg, pk)
		if err != nil {
			log.Printf("got error while preparing signed response %+v", err)
		}
		_, err = conn.Write(message)
		if err != nil {
			log.Printf("got error while writing signed response %+v", err)
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

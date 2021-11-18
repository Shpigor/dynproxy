package main

import (
	"flag"
	"log"
	"net"
	"sync"
)

var serverIp string
var serverPort int

func init() {
	flag.StringVar(&serverIp, "ip", "10.0.0.81", "listening ip address.")
	flag.IntVar(&serverPort, "p", 3030, "listening port.")
	flag.Parse()
}

func main() {
	address := &net.TCPAddr{
		IP:   net.ParseIP(serverIp),
		Port: serverPort,
	}
	group := &sync.WaitGroup{}
	group.Add(1)
	tcpServer(group, address)
	group.Wait()
}

func tcpServer(group *sync.WaitGroup, address *net.TCPAddr) {
	tcp, err := net.ListenTCP("tcp", address)
	if err != nil {
		group.Done()
		log.Fatalf("got error while listening socket: %+v", err)
	}
	go handleTcpSession(tcp)
}

func handleTcpSession(ln *net.TCPListener) {
	for {
		accept, err := ln.Accept()
		if err != nil {
			log.Printf("got error while eccepting tcp session: %+v", err)
			break
		}
		go readConn(accept)
	}
	log.Println("finished to accepting tcp connections")
}

func readConn(conn net.Conn) {
	bb := make([]byte, 2048)
	for {
		read, err := conn.Read(bb)
		if err != nil {
			if err.Error() != "EOF" {
				log.Printf("got error while reading data from connection %+v", err)
			}
			break
		}
		log.Println(">> " + string(bb[:read]))
	}
}
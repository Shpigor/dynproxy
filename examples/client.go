package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"golang.org/x/crypto/ocsp"
	"io/ioutil"
	"log"
	"net"
	"time"
)

var useTls bool
var validateOcsp bool
var address string
var caCertPath string
var certPath string
var keyPath string

var caCert *x509.Certificate
var certPool *x509.CertPool

func init() {
	var err error
	flag.BoolVar(&useTls, "t", true, "use TLS connection.")
	flag.BoolVar(&validateOcsp, "o", false, "validate OCSP response.")
	flag.StringVar(&address, "a", "10.0.0.81:2030", "connection address to the nginx.")
	flag.StringVar(&caCertPath, "ca", "/home/igor/ca/ca-cert.crt", "path to ca certificate file.")
	flag.StringVar(&certPath, "c", "/home/igor/ca/client.crt", "path to certificate file.")
	flag.StringVar(&keyPath, "k", "/home/igor/ca/client.pk", "path to private key file.")
	flag.Parse()
	certPool = x509.NewCertPool()
	caCert, err = parseCertFile(caCertPath)
	if err != nil {
		log.Fatalf("can't parse ca certificate file.")
	}
	certPool.AddCert(caCert)
}

func main() {
	for i := 0; i < 5; i++ {
		conn, err := openConnection()
		if err != nil {
			log.Fatalf("got error while connecting to tcp server: %+v", err)
		}
		message := fmt.Sprintf("Hello: %d", i)
		_, err = conn.Write([]byte(message))
		if err != nil {
			log.Fatalf("got error while writing to tcp server: %+v", err)
		}
		time.Sleep(time.Second * 2)

		err = conn.Close()
		if err != nil {
			log.Printf("got error while closing connection to tcp server: %+v", err)
		}
	}
}

func openConnection() (net.Conn, error) {
	if useTls {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			log.Fatalf("got error while parsing private key: %+v", err)
		}
		return tls.Dial("tcp", address, &tls.Config{
			Certificates:          []tls.Certificate{cert},
			RootCAs:               certPool,
			VerifyPeerCertificate: validateServerCert,
			VerifyConnection:      verifyConnection,
		})
	} else {
		return net.Dial("tcp", address)
	}
}

func verifyConnection(state tls.ConnectionState) error {
	if validateOcsp && state.OCSPResponse != nil {
		log.Printf("Verifying peer connection: %+v", string(state.OCSPResponse))
		_, err := ocsp.ParseResponse(state.OCSPResponse, caCert)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateServerCert(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	serverCert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		log.Printf("Verifying peer certificate error: %+v", err)
	}
	log.Printf("Verifying peer certificate: %+v", serverCert)
	return nil
}

func parseCertFile(filename string) (*x509.Certificate, error) {
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

func verify() {

	// To verify the signature, we provide the public key, the hashing algorithm
	// the hash sum of our message and the signature we generated previously
	// there is an optional "options" parameter which can omit for now
	err = rsa.VerifyPSS(&publicKey, crypto.SHA256, msgHashSum, signature, nil)
	if err != nil {
		fmt.Println("could not verify signature: ", err)
		return
	}
}

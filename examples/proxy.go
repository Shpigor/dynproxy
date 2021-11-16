package main

import (
	"context"
	"dynproxy"
	"log"
)

func main() {
	mainContext := context.Background()
	frCtx, _ := context.WithCancel(mainContext)
	frontend := dynproxy.Frontend{
		Context: frCtx,
		Net:     "tcp",
		Address: "0.0.0.0:2030",
		Name:    "test1",
		TlsConfig: &dynproxy.TlsConfig{
			SkipVerify: false,
			CACertPath: "/home/igor/ca/ca-cert.crt",
			CertPath:   "/home/igor/ca/server.crt",
			PkPath:     "/home/igor/ca/server.pk",
		},
	}
	err := frontend.Listen()
	if err != nil {
		log.Fatalf("error: %+v", err)
	}
	<-mainContext.Done()
}

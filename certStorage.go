package main

import (
	"crypto/tls"
	"log"
	"os"
	"sync"
)

type CertStorage struct {
	certs sync.Map
}

func (tcs *CertStorage) Fetch(hostname string, gen func() (*tls.Certificate, error)) (*tls.Certificate, error) {
	var cert tls.Certificate
	icert, ok := tcs.certs.Load(hostname)
	if ok {
		cert = icert.(tls.Certificate)
	} else {
		certp, err := gen()
		if err != nil {
			return nil, err
		}
		cert = *certp
		tcs.certs.Store(hostname, cert)
	}
	return &cert, nil
}

func NewCertStorage() *CertStorage {
	tcs := &CertStorage{}
	tcs.certs = sync.Map{}

	return tcs
}

func CAFileOepn() []byte {
	file, err := os.Open("./certification/server.crt")
	if err != nil {
		log.Fatalln("error:", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatalln("error:", err)
	}

	buffer := make([]byte, fileInfo.Size())
	_, err = file.Read(buffer)
	if err != nil {
		log.Fatalln("error:", err)
	}
	return buffer
}

func PrivateKeyFileOepn() []byte {
	file, err := os.Open("./certification/server.key")
	if err != nil {
		log.Fatalln("error:", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatalln("error:", err)
	}

	buffer := make([]byte, fileInfo.Size())
	_, err = file.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}
	return buffer
}

var caCert = CAFileOepn()

var caKey = PrivateKeyFileOepn()

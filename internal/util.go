package quicnet

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"

	"github.com/lucas-clemente/quic-go"
)

type Handler func(Ctx) error

// Setup a bare-bones TLS config for the server
func getTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"some-proto"},
	}
}

func handleMsg(conn quic.Connection, handler Handler) error {
	for {
		data, err := conn.ReceiveMessage()
		if err != nil {
			return err
		}
		err = handler(Ctx{
			Connection: conn,
			Data:       data,
		})
		if err != nil {
			return err
		}
	}
}

func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	return name
}

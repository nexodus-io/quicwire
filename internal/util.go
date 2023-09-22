package quicwire

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/quic-go/quic-go"
	"github.com/songgao/water"
)

// Handler is a function that processes incoming packets
type Handler func(packetContext) error

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

func handleMsg(tunIP *water.Interface, conn quic.Connection, handler Handler) error {
	for {
		data, err := conn.ReceiveMessage()
		if err != nil {
			return err
		}
		err = handler(packetContext{
			localIf:    tunIP,
			Connection: conn,
			Data:       data,
		})
		if err != nil {
			return err
		}
	}
}

// RetryOperation retries the operation with a backoff policy.
func RetryOperation(ctx context.Context, wait time.Duration, retries int, operation func() error) error {
	bo := backoff.WithMaxRetries(
		backoff.NewConstantBackOff(wait),
		uint64(retries),
	)
	bo = backoff.WithContext(bo, ctx)
	err := backoff.Retry(operation, bo)

	return err
}

func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	return name
}

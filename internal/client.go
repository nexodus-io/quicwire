package quicnet

import (
	"crypto/tls"
	"encoding/json"
	"fmt"

	"github.com/quic-go/quic-go"
	"github.com/songgao/water"
)

type Client struct {
	Addr    string
	Handler Handler
	TunnelInterface *water.Interface
	quic.Connection

}

func NewClient(addr string, tunIface *water.Interface) *Client {
	return &Client{
		Addr: addr,
		TunnelInterface: tunIface,}
}

func (s *Client) SetHandler(handler Handler) {
	s.Handler = handler
}

func (s *Client) Dial() error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"some-proto"},
	}
	conn, err := quic.DialAddr(s.Addr, tlsConf, &quic.Config{
		EnableDatagrams: true,
	})
	if err != nil {
		return err
	}
	s.Connection = conn
	if s.Handler != nil {
		go func() {
			err := handleMsg(s.TunnelInterface, conn, s.Handler)
			if err != nil {
				fmt.Printf("handler err: %v", err)
			}
		}()
	}
	return nil
}

func (s *Client) Send(data string) error {
	err := s.SendMessage([]byte(data))
	return err
}

func (s *Client) SendBytes(data []byte) error {
	err := s.SendMessage(data)
	return err
}

func (s *Client) SendJson(data any) error {
	res, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.SendMessage(res)
}

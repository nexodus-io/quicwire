package quicmesh

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"

	"github.com/quic-go/quic-go"
	"github.com/songgao/water"
	"go.uber.org/zap"
)

type Client struct {
	Addr            string
	localip         net.IP
	localport       int
	Handler         Handler
	TunnelInterface *water.Interface
	connection      quic.Connection
	logger          *zap.SugaredLogger
}

func NewClient(addr string, localip string, localport int, tunIface *water.Interface, logger *zap.SugaredLogger) *Client {

	ipAddr := net.ParseIP(localip)

	if ipAddr == nil {
		logger.Fatalf("Failed to parse IP address %s", localip)
	}
	return &Client{
		Addr:            addr,
		localip:         ipAddr,
		localport:       localport,
		TunnelInterface: tunIface,
		logger:          logger,
	}
}

func (c *Client) AttachHandler(handler Handler) {
	c.Handler = handler
	go func() {
		err := handleMsg(c.TunnelInterface, c.connection, c.Handler)
		if err != nil {
			fmt.Printf("handler err: %v", err)
		}
	}()
}

func (c *Client) SetConnection(conn quic.Connection) {
	c.connection = conn
}

func (c *Client) Dial() error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"some-proto"},
	}

	// udpAddr, err := net.ResolveUDPAddr("udp", c.Addr)
	// if err != nil {
	// 	return err
	// }

	// udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: c.localip, Port: 0})
	// if err != nil {
	// 	return err
	// }

	// conn, err := quic.Dial(udpConn, udpAddr, c.Addr, tlsConf, &quic.Config{
	// 	MaxIdleTimeout:  60,
	// 	KeepAlivePeriod: 30,
	// 	EnableDatagrams: true,
	// })

	conn, err := quic.DialAddr(c.Addr, tlsConf, &quic.Config{
		KeepAlivePeriod: 10,
		EnableDatagrams: true,
	})

	if err != nil {
		return err
	}
	c.connection = conn
	return nil
}

func (c *Client) Send(data string) error {
	if c.connection == nil {
		return fmt.Errorf("Client has no active connection to peer %s", c.Addr)
	}
	err := c.connection.SendMessage([]byte(data))
	return err
}

func (c *Client) SendBytes(data []byte) error {
	if c.connection == nil {
		return fmt.Errorf("Client has no active connection to peer %s", c.Addr)
	}
	err := c.connection.SendMessage(data)
	return err
}

func (c *Client) SendJson(data any) error {
	if c.connection == nil {
		return fmt.Errorf("Client has no active connection to peer %s", c.Addr)
	}
	res, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.connection.SendMessage(res)
}

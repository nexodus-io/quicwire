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

// Client struct holds state need to enable connectivity to peer
type Client struct {
	addr            string
	localip         net.IP
	localport       int
	handler         Handler
	tunnelInterface *water.Interface
	connection      quic.Connection
	logger          *zap.SugaredLogger
}

// NewClient creates a new client
func NewClient(addr string, localip string, localport int, tunIface *water.Interface, logger *zap.SugaredLogger) *Client {

	ipAddr := net.ParseIP(localip)

	if ipAddr == nil {
		logger.Fatalf("Failed to parse IP address %s", localip)
	}
	return &Client{
		addr:            addr,
		localip:         ipAddr,
		localport:       localport,
		tunnelInterface: tunIface,
		logger:          logger,
	}
}

// AttachHandler attaches a handler to process incoming packets
func (c *Client) AttachHandler(handler Handler) {
	c.handler = handler
	go func() {
		err := handleMsg(c.tunnelInterface, c.connection, c.handler)
		if err != nil {
			fmt.Printf("handler err: %v", err)
		}
	}()
}

// SetConnection sets the currently active connection to the peer
func (c *Client) SetConnection(conn quic.Connection) {
	c.connection = conn
}

// Dial establishes a connection to the peer
func (c *Client) Dial() error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"some-proto"},
	}

	// udpAddr, err := net.ResolveUDPAddr("udp", c.addr)
	// if err != nil {
	// 	return err
	// }

	// udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: c.localip, Port: 0})
	// if err != nil {
	// 	return err
	// }

	// conn, err := quic.Dial(udpConn, udpAddr, c.addr, tlsConf, &quic.Config{
	// 	KeepAlivePeriod: 10,
	// 	EnableDatagrams: true,
	// })

	conn, err := quic.DialAddr(c.addr, tlsConf, &quic.Config{
		KeepAlivePeriod: 10,
		EnableDatagrams: true,
	})

	if err != nil {
		return err
	}
	c.connection = conn
	return nil
}

// Send converts string to byte array and sends it to the peer
func (c *Client) Send(data string) error {
	if c.connection == nil {
		return fmt.Errorf("Client has no active connection to peer %s", c.addr)
	}
	err := c.connection.SendMessage([]byte(data))
	return err
}

// SendBytes sends byte array to the peer
func (c *Client) SendBytes(data []byte) error {
	if c.connection == nil {
		return fmt.Errorf("Client has no active connection to peer %s", c.addr)
	}
	err := c.connection.SendMessage(data)
	return err
}

// SendJSON converts data to json and sends it to the peer
func (c *Client) SendJSON(data any) error {
	if c.connection == nil {
		return fmt.Errorf("Client has no active connection to peer %s", c.addr)
	}
	res, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.connection.SendMessage(res)
}

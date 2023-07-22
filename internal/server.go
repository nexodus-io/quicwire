package quicmesh

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/quic-go/quic-go"
	"github.com/songgao/water"
	"go.uber.org/zap"
)

// Server struct holds state related to the server instance and its connections
type Server struct {
	addr            string
	tunnelInterface *water.Interface
	handler         Handler
	logger          *zap.SugaredLogger
}

// NewServer creates a new server that listen on given port for incoming QUIC connections
func NewServer(addr string, tunIface *water.Interface, logger *zap.SugaredLogger) *Server {
	return &Server{
		addr:            addr,
		tunnelInterface: tunIface,
		logger:          logger,
	}
}

// SetHandler sets the handler to process incoming packets
func (s *Server) SetHandler(handler Handler) {
	s.handler = handler
}

// StartServer starts the server and listens for incoming connections
func (s *Server) StartServer(ctx context.Context, qm *QuicMesh, wg *sync.WaitGroup) error {
	// Split the host and port in s.addr
	_, port, _ := net.SplitHostPort(s.addr)
	addr := fmt.Sprintf(":%s", port)
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		return err
	}
	listener, err := quic.Listen(conn, getTLSConfig(), &quic.Config{
		KeepAlivePeriod: 10,
		EnableDatagrams: true,
	})
	if err != nil {
		return err
	}

	wg.Done()

	for {
		conn, _ := listener.Accept(ctx)
		s.logger.Infof("Accepted connection from %v and local address is %v", conn.RemoteAddr(), conn.LocalAddr())
		//split host and port
		host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			return err
		}

		qm.connections[host] = conn

		// Set the client entry for the allowed ip of the host
		for _, peer := range qm.qc.peers {
			if peer.endpoint == host {
				qm.clients[peer.allowedIPs[0]] = NewClient(host, qm.qc.nodeInterface.localNodeIP, qm.qc.nodeInterface.listenPort, qm.localIf, s.logger)
				qm.clients[host].SetConnection(conn)
			}
		}

		if err != nil {
			return err
		}
		go func() {
			err := handleMsg(s.tunnelInterface, conn, s.handler)
			if err != nil {
				fmt.Printf("handler err: %v", err)
			}
		}()
	}
}

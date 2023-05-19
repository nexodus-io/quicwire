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

type Server struct {
	Addr            string
	TunnelInterface *water.Interface
	Handler         Handler
	logger          *zap.SugaredLogger
}

func NewServer(addr string, tunIface *water.Interface, logger *zap.SugaredLogger) *Server {
	return &Server{
		Addr:            addr,
		TunnelInterface: tunIface,
		logger:          logger,
	}
}

func (s *Server) SetHandler(handler Handler) {
	s.Handler = handler
}

func (s *Server) StartServer(ctx context.Context, connections map[string]quic.Connection, wg *sync.WaitGroup) error {
	listener, err := quic.ListenAddr(s.Addr, getTLSConfig(), &quic.Config{
		KeepAlivePeriod: 10,
		EnableDatagrams: true,
	})
	if err != nil {
		return err
	}

	wg.Done()

	for {
		conn, err := listener.Accept(ctx)
		s.logger.Infof("Accepted connection from %v and local address is %v", conn.RemoteAddr(), conn.LocalAddr())
		//split host and port
		host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			return err
		}

		connections[host] = conn
		if err != nil {
			return err
		}
		go func() {
			err := handleMsg(s.TunnelInterface, conn, s.Handler)
			if err != nil {
				fmt.Printf("handler err: %v", err)
			}
		}()
	}
}

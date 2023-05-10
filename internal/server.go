package quicnet

import (
	"context"
	"fmt"

	"github.com/quic-go/quic-go"
	"github.com/songgao/water"
)

type Server struct {
	Addr            string
	TunnelInterface *water.Interface
	Handler         Handler
}

func NewServer(addr string, tunIface *water.Interface) *Server {
	return &Server{
		Addr:            addr,
		TunnelInterface: tunIface,
	}
}

func (s *Server) SetHandler(handler Handler) {
	s.Handler = handler
}

func (s *Server) StartServer(ctx context.Context) error {
	listener, err := quic.ListenAddr(s.Addr, getTLSConfig(), &quic.Config{
		EnableDatagrams: true,
	})
	if err != nil {
		return err
	}
	for {
		conn, err := listener.Accept(ctx)
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

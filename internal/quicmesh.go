package quicmesh

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/songgao/water"
	"go.uber.org/zap"
)

const (
	retryInterval = 5 * time.Second
	retries       = 10
)

type QuicMesh struct {
	qc         *QuicConf
	logger     *zap.SugaredLogger
	configFile string

	// QuicNet state data
	localIf *water.Interface

	connections map[string]quic.Connection
	clients     map[string]*Client
}

func NewQuicMesh(logger *zap.SugaredLogger,
	configFile string) (*QuicMesh, error) {

	qn := &QuicMesh{
		qc:          &QuicConf{},
		logger:      logger,
		configFile:  configFile,
		connections: make(map[string]quic.Connection),
		clients:     make(map[string]*Client),
	}
	return qn, nil
}

func (qn *QuicMesh) Start(ctx context.Context, wg *sync.WaitGroup) error {
	qn.logger.Info("QuicMesh Starting")
	qn.logger.Infof("Read the quic config file : %s", qn.configFile)
	err := readQuicConf(qn.qc, qn.configFile)
	if err != nil {
		return err
	}
	qn.logger.Debugf("QuicMesh config: %v", qn.qc)
	qn.logger.Info("Create tunnel interface on local host")
	if err := qn.createTunIface(); err != nil {
		return err
	}

	// Start the server
	qn.setupTunnel(wg)
	qn.enableTrafficForwarding()
	return nil
}

func (qn *QuicMesh) Stop() {
	qn.logger.Info("QuicMesh Stop")
}

func (qn *QuicMesh) createTunIface() error {
	// Create a TUN interface
	iface, err := water.New(water.Config{DeviceType: water.TUN})
	if err != nil {
		return fmt.Errorf("Failed to create TUN interface: %w", err)
	}
	qn.logger.Debugf("TUN interface created: %s", iface.Name())

	// Assign an IP address to the TUN interface
	tunnelIpStr := fmt.Sprintf("%s/24", qn.qc.nodeInterface.localEndpoint)
	cmd := exec.Command("ip", "addr", "add", tunnelIpStr, "dev", iface.Name())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to assign IP address to TUN interface: %w", err)
	}
	qn.logger.Debugf("IP address assigned to TUN interface")

	// Up the TUN interface
	cmd = exec.Command("ip", "link", "set", "dev", iface.Name(), "up")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to change the state to UP for the TUN interface: %v", err)
	}
	qn.logger.Debugf("TUN interface %s is up and running", iface.Name())
	qn.localIf = iface

	return nil
}

func (qn *QuicMesh) setupTunnel(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		// server mode
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		localipPortStr := fmt.Sprintf("%s:%d", qn.qc.nodeInterface.localNodeIp, qn.qc.nodeInterface.listenPort)
		qn.logger.Infof("Starting server on %s", localipPortStr)
		s := NewServer(localipPortStr, qn.localIf, qn.logger)
		s.SetHandler(func(c Ctx) error {
			msg := c.Data
			qn.logger.Debugf("Client [ %s ] sent a message [ %v ] over client initiated connection", c.RemoteAddr().String(), msg)
			c.localIf.Write(c.Data)
			return nil
		})
		qn.logger.Fatal(s.StartServer(ctx, qn.connections, wg))
	}()

	wg.Wait()

	//range over all peers and create client connections
	for _, peer := range qn.qc.peers {
		qn.logger.Debugf("Starting client for peer %s", peer.endpoint)
		go func(peer Peer) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			c := NewClient(peer.endpoint, qn.qc.nodeInterface.localNodeIp, qn.qc.nodeInterface.listenPort, qn.localIf, qn.logger)

			//split endpoint to get ip and port
			host, _, err := net.SplitHostPort(peer.endpoint)
			if err != nil {
				qn.logger.Fatalf("Failed to split host and port: %v", err)
			}

			err = RetryOperation(ctx, retryInterval, retries, func() error {
				if conn, ok := qn.connections[host]; ok {
					qn.logger.Infof("Connection already exists for peer endpoint %s", peer.endpoint)
					c.SetConnection(conn)
					return nil
				} else {
					qn.logger.Debugf("No existing connection to the peer endpoint %s.", peer.endpoint)

					err := c.Dial()
					if err != nil {
						qn.logger.Debugf("Failed to dial: %v", err)
						qn.logger.Warnf("Retrying to dial %s", peer.endpoint)
						return err
					} else {
						qn.logger.Infof("Dialed new connection to peer endpoint %s.", peer.endpoint)
						c.AttachHandler(func(c Ctx) error {
							msg := c.Data
							qn.logger.Debugf("Client [ %s ] sent a message [ %v ] over server initiated connection", c.RemoteAddr().String(), msg)
							c.localIf.Write(c.Data)
							return nil
						})
						return nil
					}
				}
			})
			if err != nil {
				qn.logger.Fatalf("Peer is not reachable or : %v", err)
			}
			qn.clients[peer.allowedIPs[0]] = c
		}(peer)
	}
}

func (qn *QuicMesh) enableTrafficForwarding() error {
	go func() error {
		// Start reading packets from the TUN interface
		packet := make([]byte, 1500)
		for {
			n, err := qn.localIf.Read(packet)
			if err != nil {
				qn.logger.Fatalf("Failed to read packet from TUN interface: %v", err)
				panic(err)
			}

			dstIp := net.IP(packet[16:20])

			// Do something with the packet
			qn.logger.Debugf("Received packet from local tun interface: %v for destination %s", packet[:n], dstIp.String())

			//check if dstIp is in the list of peers
			if c, ok := qn.clients[dstIp.String()]; ok {
				err = c.SendBytes(packet[:n])
				if err != nil {
					qn.logger.Errorf("failed to send client message: %v", err)
				}
			} else {
				qn.logger.Debugf("No client connection found for destination IP %s", dstIp.String())
			}
		}
	}()
	return nil
}

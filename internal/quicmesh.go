package quicmesh

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/songgao/water"
	"go.uber.org/zap"
)

const (
	retryInterval = 5 * time.Second
	retries       = 10
	tunDevMTU     = 1190
)

type packetContext struct {
	localIf *water.Interface
	quic.Connection
	Data []byte
}

// QuicMesh struct holds state need to enable connectivity to peers
type QuicMesh struct {
	qc         *QuicConf
	logger     *zap.SugaredLogger
	configFile string

	// QuicNet state data
	localIf *water.Interface

	//NAT port binding determined through stun request
	portBinding string

	//Flag to indicate if node is behind Symmetric NAT
	symmetricNAT bool

	connections   map[string]quic.Connection
	clients       map[string]*Client
	disableClient bool
	disableServer bool
}

// NewQuicMesh creates a new QuicMesh
func NewQuicMesh(logger *zap.SugaredLogger,
	configFile string,
	disableClient bool,
	disableServer bool) (*QuicMesh, error) {

	qn := &QuicMesh{
		qc:            &QuicConf{},
		logger:        logger,
		configFile:    configFile,
		connections:   make(map[string]quic.Connection),
		clients:       make(map[string]*Client),
		disableClient: disableClient,
		disableServer: disableServer,
	}
	return qn, nil
}

// Start Initializes the QuicMesh network
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

	//find port binding
	if !qn.disableServer {
		qn.findPortBinding()
	}

	// Start the server
	qn.setupTunnel(wg, qn.disableClient, qn.disableServer)

	qn.enableTrafficForwarding()
	return nil
}

// Stop stops the QuicMesh network
func (qn *QuicMesh) Stop() {
	qn.logger.Info("QuicMesh Stop")
}

func (qn *QuicMesh) createTunIface() error {
	// Create a TUN interface
	iface, err := water.New(water.Config{DeviceType: water.TUN})
	if err != nil {
		return fmt.Errorf("failed to create Tun interface: %w", err)
	}
	qn.logger.Debugf("TUN interface created: %s", iface.Name())

	// Assign an IP address to the TUN interface
	var tunnelIPStr string
	ip, ipNet, err := net.ParseCIDR(qn.qc.nodeInterface.localEndpoint)
	if err != nil {
		// If it's not a CIDR, treat it as a plain IP supplied and append a /24 as a default prefix for convenience
		ip = net.ParseIP(qn.qc.nodeInterface.localEndpoint)
		if ip == nil {
			return fmt.Errorf("invalid IP address format: %s", qn.qc.nodeInterface.localEndpoint)
		}
		tunnelIPStr = fmt.Sprintf("%s/24", ip.String())
	} else {
		// If a mask was supplied honor it here
		ones, _ := ipNet.Mask.Size()
		tunnelIPStr = fmt.Sprintf("%s/%d", ip.String(), ones)
	}

	cmd := exec.Command("ip", "addr", "add", tunnelIPStr, "dev", iface.Name())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to assign IP address to TUN interface: %w", err)
	}
	qn.logger.Debugf("IP address assigned to TUN interface")

	// Set the MTU
	tunDevMTUString := strconv.Itoa(tunDevMTU)
	cmd = exec.Command("ip", "link", "set", "dev", iface.Name(), "mtu", tunDevMTUString)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set the MTU: %v", err)
	}

	// Up the TUN interface
	cmd = exec.Command("ip", "link", "set", "dev", iface.Name(), "up")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to change the state to UP for the TUN interface: %v", err)
	}

	qn.logger.Debugf("TUN interface %s is up and running", iface.Name())
	qn.localIf = iface

	return nil
}

func (qn *QuicMesh) findPortBinding() (string, error) {

	isSymmetric, err := IsSymmetricNAT(qn.qc.nodeInterface.listenPort)
	if err != nil {
		qn.logger.Error(err)
	}
	if isSymmetric {
		qn.logger.Warn("Node is behind Symmetric NAT")
		return "", fmt.Errorf("node is behind Symmetric NAT")
	}

	res, err := GetPortBinding(qn.qc.nodeInterface.listenPort)
	if err != nil {
		qn.logger.Fatalf("stun request failed: %v", err)
	}
	qn.logger.Infof("Port binding returned by STUN request: %s", res)
	return res, nil
}

func (qn *QuicMesh) setupTunnel(wg *sync.WaitGroup, disableClient bool, disableServer bool) {
	// Create a shared UDP socket
	localipPortStr := fmt.Sprintf("%s:%d", qn.qc.nodeInterface.localNodeIP, qn.qc.nodeInterface.listenPort)
	udpAddr, err := net.ResolveUDPAddr("udp4", localipPortStr)
	if err != nil {
		qn.logger.Fatalf("Failed to resolve UDP address: %v", err)
	}
	udpConn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		qn.logger.Fatalf("Failed to create shared UDP socket: %v", err)
	}

	if !disableServer {
		wg.Add(1)
		go func() {
			// server mode
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			qn.logger.Infof("Starting server on %s", localipPortStr)
			s := NewServer(localipPortStr, qn.localIf, qn.logger)
			s.SetHandler(func(c packetContext) error {
				msg := c.Data
				qn.logger.Debugf("Client [ %s ] sent a message [ %v ] over client initiated connection", c.RemoteAddr().String(), msg)
				c.localIf.Write(c.Data)
				return nil
			})
			qn.logger.Fatal(s.StartServer(ctx, udpConn, qn, wg))
		}()
		wg.Wait()
	}

	if !disableClient {

		//range over all peers and create client connections
		for _, peer := range qn.qc.peers {
			qn.logger.Debugf("Starting client for peer %s", peer.endpoint)
			go func(peer Peer) {

				_, ok := qn.clients[peer.allowedIPs[0]]
				if ok {
					qn.logger.Infof("Client already exists for peer %s [ %s ]", peer.endpoint, peer.allowedIPs[0])
					return
				}

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				c := NewClient(peer.endpoint, qn.qc.nodeInterface.localNodeIP, qn.qc.nodeInterface.listenPort, qn.localIf, qn.logger)

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
					}
					qn.logger.Debugf("No existing connection to the peer endpoint %s.", peer.endpoint)

					err := c.Dial(udpConn)
					if err != nil {
						qn.logger.Debugf("Failed to dial: %v", err)
						qn.logger.Warnf("Retrying to dial %s", peer.endpoint)
						return err
					}
					qn.logger.Infof("Dialed new connection to peer endpoint %s.", peer.endpoint)
					c.AttachHandler(func(c packetContext) error {
						msg := c.Data
						qn.logger.Debugf("Client [ %s ] sent a message [ %v ] over server initiated connection", c.RemoteAddr().String(), msg)
						c.localIf.Write(c.Data)
						return nil
					})
					return nil
				})
				if err != nil {
					qn.logger.Warnf("Peer %s is not reachable or : %v", peer.endpoint, err)
				}
				qn.clients[peer.allowedIPs[0]] = c
			}(peer)
		}
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

			dstIP := net.IP(packet[16:20])

			// Do something with the packet
			qn.logger.Debugf("Received packet from local tun interface: %v for destination %s", packet[:n], dstIP.String())

			//check if dstIp is in the list of peers
			if c, ok := qn.clients[dstIP.String()]; ok {
				err = c.SendBytes(packet[:n])
				if err != nil {
					qn.logger.Errorf("failed to send client message: %v", err)
				}
				//check if dstIp is in the Con
			} else {
				qn.logger.Debugf("No client connection found for destination IP %s", dstIP.String())
			}
		}
	}()
	return nil
}

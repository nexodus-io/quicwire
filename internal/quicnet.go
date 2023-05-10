package quicnet

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/songgao/water"
	"go.uber.org/zap"
)

type QuicNet struct {
	qc         *QuicConf
	logger     *zap.SugaredLogger
	configFile string
	isServer   bool
	isClient   bool

	// QuicNet state data
	localIf *water.Interface
}

func NewQuicNet(logger *zap.SugaredLogger,
	configFile string,
	isServer bool,
	isClient bool) (*QuicNet, error) {

	qn := &QuicNet{
		qc:         &QuicConf{},
		logger:     logger,
		configFile: configFile,
		isServer:   isServer,
		isClient:   isClient,
	}
	return qn, nil
}

func (qn *QuicNet) Start(ctx context.Context, wg *sync.WaitGroup) error {
	qn.logger.Info("QuicNet Starting")
	qn.logger.Infof("Read the quic config file : %s", qn.configFile)
	err := readQuicConf(qn.qc, qn.configFile)
	if err != nil {
		return err
	}

	qn.logger.Info("Trying to create tunnel interface on local host")
	if err := qn.createTunIface(); err != nil {
		return err
	}

	// Start the server
	qn.setupTunnel(wg)
	return nil
}

func (qn *QuicNet) Stop() {
	qn.logger.Info("QuicNet Stop")
}

func (qn *QuicNet) createTunIface() error {
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

func (qn *QuicNet) setupTunnel(wg *sync.WaitGroup) {

	if qn.isServer {
		go func() {
			// server mode
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			localipPortStr := fmt.Sprintf("%s:%d", qn.qc.nodeInterface.localNodeIp, qn.qc.nodeInterface.listenPort)
			qn.logger.Infof("Starting server on %s", localipPortStr)

			s := NewServer(localipPortStr, qn.localIf)
			s.SetHandler(func(c Ctx) error {
				msg := c.Data
				qn.logger.Debugf("Client [ %s ] sent a message [ %v ]", c.RemoteAddr().String(), msg)
				c.localIf.Write(c.Data)
				return nil
			})
			qn.logger.Fatal(s.StartServer(ctx))
		}()
	}

	if qn.isClient {
		peer := qn.qc.peers[0]
		go func() {
			_, cancel := context.WithCancel(context.Background())
			defer cancel()
			c := NewClient(peer.endpoint, qn.localIf)
			//write while loop to call Dial till condition becomes true
			retries := 0
			for {
				err := c.Dial()
				if err != nil {
					qn.logger.Warnf("Failed to dial: %v", err)
					qn.logger.Warnf("Retrying to dial %s", peer.endpoint)
					retries++
				} else {
					break
				}
				if retries > 5 {
					break
				}
				time.Sleep(10 * time.Second)
			}

			// Start reading packets from the TUN interface
			packet := make([]byte, 1500)
			for {
				n, err := qn.localIf.Read(packet)
				if err != nil {
					qn.logger.Fatalf("Failed to read packet from TUN interface: %v", err)
					panic(err)
				}

				// Do something with the packet
				qn.logger.Debugf("Received packet: %v", packet[:n])
				err = c.SendBytes(packet[:n])
				if err != nil {
					qn.logger.Errorf("failed to send client message: %v", err)
				}

			}

		}()
	}

}

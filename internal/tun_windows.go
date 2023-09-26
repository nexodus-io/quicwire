//go:build windows

package quicwire

import (
	"fmt"
	"net"
	"os/exec"

	"github.com/songgao/water"
)

func (qn *QuicWire) createTunIface() error {
	// Create a TUN interface
	iface, err := water.New(water.Config{DeviceType: water.TUN})
	if err != nil {
		return fmt.Errorf("failed to create Tun interface: %w", err)
	}
	qn.logger.Debugf("TUN interface created: %s", iface.Name())

	// Assign an IP address to the TUN interface
	ip, _, err := net.ParseCIDR(qn.qc.nodeInterface.localEndpoint)
	if err != nil {
		// If it's not a CIDR, treat it as a plain IP supplied and append a /24 as a default prefix for convenience
		ip = net.ParseIP(qn.qc.nodeInterface.localEndpoint)
		if ip == nil {
			return fmt.Errorf("invalid IP address format: %s", qn.qc.nodeInterface.localEndpoint)
		}
	}

	// Assign IP using netsh
	cmd := exec.Command("netsh", "interface", "ip", "set", "address", iface.Name(), "static", ip.String(), "255.255.255.0")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to assign IP address to TUN interface: %w", err)
	}
	qn.logger.Debugf("IP address assigned to TUN interface")
	qn.logger.Debugf("TUN interface %s is up and running", iface.Name())
	qn.localIf = iface

	return nil
}

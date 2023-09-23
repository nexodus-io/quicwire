//go:build linux

package quicwire

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"

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
	var tunnelIPStr string
	ip, ipNet, err := net.ParseCIDR(qn.qc.nodeInterface.localEndpoint)
	if err != nil {
		// If it's not a CIDR, append a /24 as a default prefix for convenience of not having to add a route for convenience
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

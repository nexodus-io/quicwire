//go:build darwin

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
	ip, _, err := net.ParseCIDR(qn.qc.nodeInterface.localEndpoint)
	if err != nil {
		// If it's not a CIDR, treat it as a plain IP supplied and append a /24 as a default prefix for convenience
		ip = net.ParseIP(qn.qc.nodeInterface.localEndpoint)
		if ip == nil {
			return fmt.Errorf("invalid IP address format: %s", qn.qc.nodeInterface.localEndpoint)
		}
	}

	cmd := exec.Command("ifconfig", iface.Name(), "inet", ip.String(), ip.String(), "alias")
	qn.logger.Infof("Interface %s is being assigned the address %s with remote address %s as an alias", iface.Name(), ip.String(), ip.String())
	qn.logger.Infof("Running command: %v", cmd)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to assign IP address to TUN interface: %w", err)
	}
	qn.logger.Debugf("IP address assigned to TUN interface")

	// Set the MTU
	tunDevMTUString := strconv.Itoa(tunDevMTU)
	cmd = exec.Command("ifconfig", iface.Name(), "mtu", tunDevMTUString)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set the MTU: %v", err)
	}

	// The interface is already up from the previous command, so no need to run another command to bring it up
	qn.logger.Debugf("TUN interface %s is up and running", iface.Name())
	qn.localIf = iface

	return nil
}

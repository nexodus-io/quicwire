package main

import (
	"log"
	"os/exec"

	"github.com/songgao/water"
)

func main() {
	// Create a TUN interface
	iface, err := water.New(water.Config{DeviceType: water.TUN})
	if err != nil {
		log.Fatalf("Failed to create TUN interface: %v", err)
	}
	log.Printf("TUN interface created: %v", iface.Name())

	// Assign an IP address to the TUN interface
	cmd := exec.Command("ip", "addr", "add", "192.168.168.1/24", "dev", iface.Name())
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to assign IP address to TUN interface: %v", err)
	}
	log.Printf("IP address assigned to TUN interface")

	// Start reading packets from the TUN interface
	packet := make([]byte, 1500)
	for {
		n, err := iface.Read(packet)
		if err != nil {
			log.Fatalf("Failed to read packet from TUN interface: %v", err)
		}

		// Do something with the packet
		log.Printf("Received packet: %v", packet[:n])
	}
}

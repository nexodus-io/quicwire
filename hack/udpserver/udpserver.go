package main

import (
	"fmt"
	"net"
)

func main() {
	// Create a UDP address to listen on
	addr, err := net.ResolveUDPAddr("udp4", ":55380")
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	fmt.Println("addr is ", addr.IP, addr.Port)

	// Create a UDP connection to listen for incoming packets
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		fmt.Println("Error listening on UDP:", err)
		return
	}

	//print conn4
	fmt.Println("conn is ", conn.LocalAddr(), conn.RemoteAddr())

	defer conn.Close()

	// Create a buffer to store incoming data
	buffer := make([]byte, 1500)

	fmt.Println("UDP server is running. Listening on :55380")

	// Continuously listen for incoming packets on both connections
	for {
		// Read data from the UDP4 connection into the buffer
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading from UDP4 connection:", err)
			return
		}

		// Print the received message
		fmt.Printf("Received message from %s (UDP): %s\n", addr.String(), string(buffer[:n]))

		// Send a response back to the client on the UDP4 connection
		response := []byte("Message received (UDP4)!")
		_, err = conn.WriteToUDP(response, addr)
		if err != nil {
			fmt.Println("Error sending response (UDP4):", err)
			return
		}

	}
}

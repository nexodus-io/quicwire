package main

import (
	"fmt"
	"net"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func main() {
	// Create a UDP address to connect to
	addr, err := net.ResolveUDPAddr("udp", "")
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return
	}

	fmt.Println("addr is ", addr.String())

	d := net.Dialer{
		Control:   Control,
		LocalAddr: addr,
		Timeout:   time.Duration(0),
	}

	// Create a UDP connection to the server
	//conn, err := net.DialUDP("udp", nil, addr)
	conn, err := d.Dial("udp", "172.31.28.175:55380")
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}

	fmt.Println("conn is ", conn.LocalAddr(), conn.RemoteAddr())

	defer conn.Close()

	// Send a message to the server
	message := []byte("Hello, server!")
	_, err = conn.Write(message)
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}

	//print message
	fmt.Println("message is ", string(message))

	// Create a buffer to store the response
	buffer := make([]byte, 1500)

	// Read the response from the server
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}

	// Print the response from the server
	fmt.Println("Response from server:", string(buffer[:n]))
}

// Control is a Dialer.Control function for setting SO_REUSEADDR and SO_REUSEPORT.
func Control(network, address string, c syscall.RawConn) (err error) {
	controlErr := c.Control(func(fd uintptr) {
		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
		if err != nil {
			return
		}
		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	})
	if controlErr != nil {
		err = controlErr
	}
	return
}

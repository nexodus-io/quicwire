package quicmesh

import (
	"fmt"
	"net"
	"strconv"

	"github.com/libp2p/go-reuseport"
	"github.com/pion/stun"
	log "github.com/sirupsen/logrus"
)

const (
	stunServer1 = "stun1.l.google.com:19302"
	stunServer2 = "stun2.l.google.com:19302"
)

// IsSymmetricNAT attempts to infer if the node is behind a symmetric
// nat device by querying two STUN servers. If the requests return
// different ports, then it is likely the node is behind a symmetric nat.
func IsSymmetricNAT(sourcePort int) (bool, error) {
	firstStun, err := StunRequest(stunServer1, sourcePort)
	if err != nil {
		return false, fmt.Errorf("failed to query the STUN server %s", stunServer1)
	}
	log.Infof("STUN Result from %s => [ %s ]", stunServer1, firstStun)
	secondStun, err := StunRequest(stunServer2, sourcePort)
	if err != nil {
		return false, fmt.Errorf("failed to query the STUN server %s", stunServer1)
	}
	if firstStun != secondStun {
		return true, nil
	}
	log.Infof("STUN Result from %s => [ %s ]", stunServer1, secondStun)

	return false, nil
}

// GetPortBinding returns the NAT port binding (IP:port) of the node
func GetPortBinding(sourcePort int) (string, error) {
	stunServer := stunServer1
	res, err := StunRequest(stunServer, sourcePort)
	if err != nil {
		return "", fmt.Errorf("stun request failed: %v", err)
	}
	return res, nil
}

// StunRequest initiate a connection to a STUN server sourced from the wg src port
func StunRequest(stunServer string, srcPort int) (string, error) {

	log.Debugf("dialing stun server %s", stunServer)

	conn, err := reuseport.Dial("udp4", fmt.Sprintf(":%d", srcPort), stunServer)
	if err != nil {
		log.Errorf("stun dialing timed out %v", err)
		return "", fmt.Errorf("failed to dial stun server %s: %w", stunServer, err)
	}

	defer conn.Close()
	stunResults, err := stunDialer(&conn)
	if err != nil {
		return "", fmt.Errorf("stun dialing timed out %w", err)
	}
	return stunResults, nil
}

func stunDialer(conn *net.Conn) (string, error) {
	c, err := stun.NewClient(*conn)
	if err != nil {
		log.Errorf("%v", err)
	}
	var xorAddr stun.XORMappedAddress
	if err = c.Do(stun.MustBuild(stun.TransactionID, stun.BindingRequest), func(res stun.Event) {
		if res.Error != nil {
			log.Println(res.Error)
			return
		}
		if getErr := xorAddr.GetFrom(res.Message); getErr != nil {
			log.Println(getErr)
			if err := c.Close(); err != nil {
				log.Println(err)
				return
			}
			return
		}
		log.Debugf("Stun address and port is: %s:%d", xorAddr.IP, xorAddr.Port)

	}); err != nil {
		return "", err
	}
	if err := c.Close(); err != nil {
		return "", err
	}

	if xorAddr.IP == nil {
		return "", fmt.Errorf("no response")
	}
	stunAddress := net.JoinHostPort(xorAddr.IP.String(), strconv.Itoa(xorAddr.Port))
	if err != nil {
		return "", err
	}

	return stunAddress, nil
}

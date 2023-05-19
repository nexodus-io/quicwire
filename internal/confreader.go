package quicmesh

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Peer represents a peer in the quicmesh configuration file
type Peer struct {
	allowedIPs          []string
	endpoint            string
	persistentKeepalive string
}

// nodeInterface represents the node interface in the quicmesh configuration file
type nodeInterface struct {
	listenPort    int
	localEndpoint string
	localNodeIP   string
}

// QuicConf contains the quicmesh configuration file data
type QuicConf struct {
	nodeInterface nodeInterface
	peers         []Peer
}

func readQuicConf(qc *QuicConf, configFile string) error {
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Variables to store values from the file
	var section, localEndpoint, localNodeIP, endpoint, persistentKeepalive string
	var listenPort int
	var allowedIPs []string

	for scanner.Scan() {
		line := scanner.Text()
		// Trim any leading or trailing spaces from the line
		line = strings.TrimSpace(line)

		// Ignore any comments or empty lines
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// Check if the line starts with a section header
		if line[0] == '[' && line[len(line)-1] == ']' {
			if section != "" && section == "Interface" {
				qc.nodeInterface.listenPort = listenPort
				qc.nodeInterface.localNodeIP = localNodeIP
				qc.nodeInterface.localEndpoint = localEndpoint
			}

			if section != "" && section == "Peer" {
				qc.peers = append(qc.peers, Peer{
					allowedIPs:          allowedIPs,
					endpoint:            endpoint,
					persistentKeepalive: persistentKeepalive,
				})
			}

			// Extract the section name and print it
			section = line[1 : len(line)-1]

			// Reset variables for new section
			allowedIPs = nil

		} else {
			// Split the line into key and value parts
			parts := strings.Split(line, " = ")
			if len(parts) != 2 {
				continue
			}

			// Extract the key and value
			key := parts[0]
			value := parts[1]

			// Store the values in the corresponding variables
			switch key {
			case "ListenPort":
				listenPort, err = strconv.Atoi(value)
				if err != nil {
					return err
				}
			case "LocalEndpoint":
				localEndpoint = value
			case "LocalNodeIp":
				localNodeIP = value
			case "AllowedIPs":
				allowedIPs = strings.Split(value, ",")
			case "Endpoint":
				endpoint = value
			case "PersistentKeepalive":
				persistentKeepalive = value
			default:
			}

		}
	}
	if section != "" && section == "Interface" {
		qc.nodeInterface.listenPort = listenPort
		qc.nodeInterface.localNodeIP = localNodeIP
		qc.nodeInterface.localEndpoint = localEndpoint
	}

	if section != "" && section == "Peer" {
		qc.peers = append(qc.peers, Peer{
			allowedIPs:          allowedIPs,
			endpoint:            endpoint,
			persistentKeepalive: persistentKeepalive,
		})
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

// SmartSocketDetector tries multiple common socket paths and validates connectivity
type SmartSocketDetector struct {
	candidatePaths  []string
	timeoutDuration time.Duration
}

func NewSmartSocketDetector() *SmartSocketDetector {
	return &SmartSocketDetector{
		candidatePaths: []string{
			// User-specified environment variable (highest priority)
			os.Getenv("CARDANO_NODE_SOCKET"),

			// Primary working socket (current installation)
			"/opt/cardano/cnode/sockets/node.socket",

			// Backup dolos socket (if dolos is running)
			"/root/workspace/cardano-stack/dolos/data/dolos.socket.sock",

			// Guild tools path (if guild container is running)
			"/root/workspace/cardano-node-guild/socket/node.socket",

			// Common Docker paths (only if in containerized environment)
			"/var/cardano/socket/node.socket",
			"/cardano/socket/node.socket",
		},
		timeoutDuration: 3 * time.Second, // Faster timeout for quicker detection
	}
}

// DetectCardanoSocket finds the best available Cardano node socket
func (ssd *SmartSocketDetector) DetectCardanoSocket() (string, error) {
	log.Println("🔍 Smart socket detection starting...")

	var workingSocket string
	var connectionErrors []string

	for _, path := range ssd.candidatePaths {
		if path == "" {
			continue // Skip empty environment variable
		}

		// Only check if socket file exists first
		if !ssd.socketExists(path) {
			continue // Silently skip non-existent sockets (no noise)
		}

		// Test actual connectivity for existing sockets
		if ssd.testConnectivity(path) {
			workingSocket = path
			log.Printf("✅ Connected to: %s", path)
			break // Found working socket, stop searching
		} else {
			// Only log connectivity issues for sockets that exist but don't work
			connectionErrors = append(connectionErrors, fmt.Sprintf("Socket exists but not responsive: %s", path))
		}
	}

	if workingSocket == "" {
		// Only now report the actual connection errors for existing sockets
		if len(connectionErrors) > 0 {
			log.Println("⚠️ Socket connectivity issues:")
			for _, err := range connectionErrors {
				log.Printf("   %s", err)
			}
		}
		return "", fmt.Errorf("no accessible Cardano node socket found")
	}

	log.Printf("🎯 Using socket: %s", workingSocket)
	return workingSocket, nil
}

// socketExists checks if socket file exists and is a socket
func (ssd *SmartSocketDetector) socketExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Check if it's a socket file
	return info.Mode()&os.ModeSocket != 0
}

// testConnectivity attempts to connect to the socket
func (ssd *SmartSocketDetector) testConnectivity(path string) bool {
	conn, err := net.DialTimeout("unix", path, ssd.timeoutDuration)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Try a simple write/read to ensure socket is responsive
	_, err = conn.Write([]byte{})
	return err == nil // Even if write fails, connection was established
}

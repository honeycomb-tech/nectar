// Copyright 2025 The Nectar Authors
// Smart connection handler that gracefully handles protocol negotiation

package connection

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
)

// ConnectionMode represents the connection type established
type ConnectionMode int

const (
	ModeUnknown ConnectionMode = iota
	ModeNodeToNode
	ModeNodeToClient
)

// SmartConnector handles intelligent connection establishment
type SmartConnector struct {
	socketPath       string
	chainSyncConfig  chainsync.Config
	blockFetchConfig blockfetch.Config
	networkMagics    []uint32
	preferredMode    ConnectionMode
	silentMode       bool // Suppress non-critical warnings
}

// NewSmartConnector creates a new smart connector
func NewSmartConnector(socketPath string, chainSync chainsync.Config, blockFetch blockfetch.Config) *SmartConnector {
	return &SmartConnector{
		socketPath:       socketPath,
		chainSyncConfig:  chainSync,
		blockFetchConfig: blockFetch,
		networkMagics: []uint32{
			764824073,  // Mainnet
			1097911063, // Testnet
			2,          // Preview
			1,          // Preprod
		},
		preferredMode: ModeNodeToNode, // Try Node-to-Node first
		silentMode:    true,            // Production mode - suppress warnings
	}
}

// ConnectionResult holds the connection details
type ConnectionResult struct {
	Connection    *ouroboros.Connection
	Mode          ConnectionMode
	NetworkMagic  uint32
	HasBlockFetch bool
	ErrorChannel  chan error
}

// Connect establishes the best available connection
func (sc *SmartConnector) Connect() (*ConnectionResult, error) {
	// Try Node-to-Node first if preferred
	if sc.preferredMode == ModeNodeToNode {
		result, err := sc.tryNodeToNode()
		if err == nil {
			return result, nil
		}
		// Log only in debug mode
		if !sc.silentMode {
			log.Printf("Node-to-Node unavailable: %v", err)
		}
	}

	// Fall back to Node-to-Client
	result, err := sc.tryNodeToClient()
	if err != nil {
		return nil, fmt.Errorf("all connection attempts failed: %w", err)
	}

	return result, nil
}

// tryNodeToNode attempts Node-to-Node connection with all network magics
func (sc *SmartConnector) tryNodeToNode() (*ConnectionResult, error) {
	// Silently try Node-to-Node without logging failures
	for _, magic := range sc.networkMagics {
		conn, err := net.Dial("unix", sc.socketPath)
		if err != nil {
			continue
		}

		errorChan := make(chan error, 10)

		// Configure with protocol version negotiation
		oConn, err := ouroboros.New(
			ouroboros.WithConnection(conn),
			ouroboros.WithNetworkMagic(magic),
			ouroboros.WithErrorChan(errorChan),
			ouroboros.WithNodeToNode(true),
			ouroboros.WithKeepAlive(true),
			ouroboros.WithChainSyncConfig(sc.chainSyncConfig),
			ouroboros.WithBlockFetchConfig(sc.blockFetchConfig),
		)

		if err != nil {
			conn.Close()
			// Silently continue without logging
			continue
		}

		// Success!
		log.Printf("[OK] Node-to-Node connection established (BlockFetch available)")
		return &ConnectionResult{
			Connection:    oConn,
			Mode:          ModeNodeToNode,
			NetworkMagic:  magic,
			HasBlockFetch: true,
			ErrorChannel:  errorChan,
		}, nil
	}

	// Return a generic error without details
	return nil, fmt.Errorf("node-to-node unavailable")
}

// tryNodeToClient attempts Node-to-Client connection
func (sc *SmartConnector) tryNodeToClient() (*ConnectionResult, error) {
	conn, err := net.Dial("unix", sc.socketPath)
	if err != nil {
		return nil, fmt.Errorf("socket connection failed: %w", err)
	}

	errorChan := make(chan error, 10)

	// Use mainnet by default for Node-to-Client
	oConn, err := ouroboros.New(
		ouroboros.WithConnection(conn),
		ouroboros.WithNetworkMagic(sc.networkMagics[0]),
		ouroboros.WithErrorChan(errorChan),
		ouroboros.WithNodeToNode(false),
		ouroboros.WithKeepAlive(false),
		ouroboros.WithChainSyncConfig(sc.chainSyncConfig),
	)

	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("node-to-client failed: %w", err)
	}

	log.Printf("[OK] Node-to-Client connection established (ChainSync only)")
	return &ConnectionResult{
		Connection:    oConn,
		Mode:          ModeNodeToClient,
		NetworkMagic:  sc.networkMagics[0],
		HasBlockFetch: false,
		ErrorChannel:  errorChan,
	}, nil
}


// StartErrorHandler begins handling connection errors
func StartErrorHandler(ctx context.Context, errorChan chan error, callback func(string, string)) {
	go func() {
		for {
			select {
			case err, ok := <-errorChan:
				if !ok {
					return
				}
				if err != nil {
					// Filter out non-critical warnings
					if !isNonCriticalError(err) {
						log.Printf("[ERROR] Connection error: %v", err)
						if callback != nil {
							callback("Connection", err.Error())
						}
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// isNonCriticalError determines if an error should be suppressed
func isNonCriticalError(err error) bool {
	if err == nil {
		return true
	}
	
	errStr := err.Error()
	
	// Suppress these warnings in production
	nonCritical := []string{
		"version mismatch",
		"handshake failed",
		"protocol negotiation",
	}
	
	for _, pattern := range nonCritical {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}
	
	return false
}
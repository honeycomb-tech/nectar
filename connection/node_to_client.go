// Copyright 2025 The Nectar Authors
// Node-to-Client only connection handler

package connection

import (
	"fmt"
	"log"
	"net"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
)

// NodeToClientConnector handles Node-to-Client connections only
type NodeToClientConnector struct {
	socketPath      string
	chainSyncConfig chainsync.Config
	networkMagic    uint32
}

// NewNodeToClientConnector creates a new Node-to-Client connector
func NewNodeToClientConnector(socketPath string, chainSync chainsync.Config) *NodeToClientConnector {
	return &NodeToClientConnector{
		socketPath:      socketPath,
		chainSyncConfig: chainSync,
		networkMagic:    getNetworkMagicFromEnv(),
	}
}

// Connect establishes a Node-to-Client connection
func (nc *NodeToClientConnector) Connect() (*ConnectionResult, error) {
	conn, err := net.Dial("unix", nc.socketPath)
	if err != nil {
		return nil, fmt.Errorf("socket connection failed: %w", err)
	}

	errorChan := make(chan error, 10)

	// Create Node-to-Client connection
	oConn, err := ouroboros.New(
		ouroboros.WithConnection(conn),
		ouroboros.WithNetworkMagic(nc.networkMagic),
		ouroboros.WithErrorChan(errorChan),
		ouroboros.WithNodeToNode(false), // Node-to-Client mode
		ouroboros.WithKeepAlive(false),
		ouroboros.WithChainSyncConfig(nc.chainSyncConfig),
	)

	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("node-to-client connection failed: %w", err)
	}

	log.Printf("[OK] Node-to-Client connection established")
	return &ConnectionResult{
		Connection:    oConn,
		Mode:          ModeNodeToClient,
		NetworkMagic:  nc.networkMagic,
		HasBlockFetch: false, // BlockFetch not available in N2C mode
		ErrorChannel:  errorChan,
	}, nil
}
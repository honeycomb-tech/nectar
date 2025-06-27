# Nectar 🍯

A Cardano blockchain indexer designed for TiDB's distributed architecture.

## ⚠️ Under Heavy Development

This project is in active development and not yet ready for production use. We're working on indexing the Cardano blockchain into TiDB to enable efficient querying of on-chain data.

## What is Nectar?

Nectar indexes Cardano blockchain data into TiDB, making it queryable via SQL. It processes all eras from Byron to Conway, storing transactions, UTXOs, stake pools, delegations, and more.

## Why TiDB?

TiDB provides horizontal scalability through its distributed architecture. Unlike traditional single-node databases, TiDB can scale by adding more TiKV nodes, making it suitable for blockchain data that grows continuously.

## Current Status

- Actively syncing Cardano mainnet
- Processing all transaction types across all eras
- Building comprehensive indexes for efficient queries
- Implementing memory optimizations and performance improvements

## Contributing

We welcome contributions! This is an open-source project and we're looking for help with:

- Performance optimizations
- Additional query indexes
- Documentation improvements
- Testing and bug reports
- Feature suggestions

Please feel free to:
- Open issues for bugs or feature requests
- Submit pull requests
- Join discussions about the project direction
- Share your use cases and requirements

## Development Setup

### Requirements

- Go 1.21+
- TiDB cluster (or TiUP playground for testing)
- Cardano node with local socket access
- 8+ CPU cores, 16GB+ RAM recommended

### Quick Start

1. Clone the repository:
```bash
git clone https://github.com/honeycomb-tech/nectar.git
cd nectar
```

2. Build:
```bash
go build -o nectar .
```

3. Configure (create `nectar.toml`):
```toml
[database]
dsn = "root:password@tcp(localhost:4000)/nectar"

[cardano]
node_socket = "/path/to/node.socket"
network_magic = 764824073  # mainnet
```

4. Run:
```bash
./nectar
```

## Architecture

Nectar uses a modular processor architecture:
- **Block Processor**: Orchestrates processing of each block
- **Transaction Processor**: Handles all transaction types
- **Certificate Processor**: Processes stake pool and delegation certificates
- **Asset Processor**: Indexes native tokens and NFTs
- **Metadata Processor**: Stores transaction metadata
- **Script Processor**: Handles Plutus scripts

See [PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md) for detailed code organization.

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.

## Contact

- GitHub Issues: [github.com/honeycomb-tech/nectar/issues](https://github.com/honeycomb-tech/nectar/issues)
- Discussions: [github.com/honeycomb-tech/nectar/discussions](https://github.com/honeycomb-tech/nectar/discussions)

---

Built with ❤️ for the Cardano community
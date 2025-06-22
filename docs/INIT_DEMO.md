# Nectar Interactive Setup Demo

## First Run Experience

When you run Nectar for the first time without a configuration file:

```bash
$ nectar
❌ Configuration file not found: nectar.toml

Please run 'nectar init' to create a configuration file.
Example:
  nectar init                    # Interactive setup
  nectar init --no-interactive   # Create default config
```

## Interactive Setup

Running `nectar init` launches an interactive wizard:

```bash
$ nectar init
🚀 Welcome to Nectar Setup Wizard
=================================

This wizard will help you create a configuration file for Nectar.
Press Enter to use the [default] values shown.

📡 Network Configuration
------------------------
Select Cardano network:
  1) Mainnet (default)
  2) Preprod
  3) Preview
  4) Custom
Choice [1]: 1
✓ Selected: Mainnet

Cardano node socket path [/opt/cardano/cnode/sockets/node.socket]: 

💾 Database Configuration
------------------------
Enter your TiDB connection details.
Database host [localhost]: 
Database port [4000]: 
Database name [nectar]: 
Database user [root]: 
Database password (will be stored in config): mypassword

⚡ Performance Configuration
---------------------------
Number of worker threads [8]: 16
Database connection pool size [8]: 16

📊 Dashboard Configuration
-------------------------
Select dashboard type:
  1) Terminal (default)
  2) Web
  3) Both
  4) None
Choice [1]: 2
Web dashboard port [8080]: 

📝 Generating Configuration
--------------------------

✅ Configuration saved to: nectar.toml

📋 Configuration Summary
------------------------
Network:        Mainnet (magic: 764824073)
Node Socket:    /opt/cardano/cnode/sockets/node.socket
Database:       root@localhost/nectar
Workers:        16
Dashboard:      web
Web Port:       8080

🎉 Setup complete! You can now run:
   nectar --config nectar.toml
```

## Non-Interactive Setup

For automated deployments, use the `--no-interactive` flag:

```bash
$ nectar init --no-interactive
Configuration file created at: nectar.toml
Please edit this file to match your environment settings.
```

This creates a template configuration that you can edit manually.

## Configuration Required

The new setup enforces configuration - no more guessing with environment variables:

```bash
$ nectar
❌ Configuration file not found: nectar.toml

Please run 'nectar init' to create a configuration file.
```

## Migration from Environment Variables

If you have an existing setup with environment variables:

```bash
$ export TIDB_DSN="root:password@tcp(localhost:4000)/nectar"
$ export WORKER_COUNT=32
$ nectar migrate-env
Configuration migrated from environment to: nectar.toml
```

## Benefits

1. **User-Friendly**: Interactive wizard guides through all settings
2. **No Fallbacks**: Clear error when config is missing
3. **Type Safety**: Validates inputs during setup
4. **Self-Documenting**: Generated config has comments
5. **Network Aware**: Pre-configured settings for each network
# Nectar Dashboard System

This is the new dashboard system for Nectar that supports both terminal and web interfaces.

## Features

- **Terminal Dashboard**: Simple progress spinner with key metrics
- **Web Dashboard**: Full-featured web interface with real-time updates
- **Dual Mode**: Run both dashboards simultaneously
- **Environment Configuration**: Easy configuration via environment variables

## Configuration

The dashboard system can be configured using environment variables:

- `DASHBOARD_TYPE`: Choose dashboard type (`terminal`, `web`, `both`, `none`)
  - Default: `both`
- `WEB_PORT`: Port for web dashboard
  - Default: `8080`
- `DETAILED_LOG`: Enable detailed terminal output
  - Default: `false`

## Usage

### Terminal Only
```bash
export DASHBOARD_TYPE=terminal
./nectar
```

### Web Only
```bash
export DASHBOARD_TYPE=web
export WEB_PORT=8080
./nectar
```

### Both (Default)
```bash
./nectar
# Access web dashboard at http://your-vm-ip:8080
```

### No Dashboard
```bash
export DASHBOARD_TYPE=none
./nectar
```

## Architecture

The dashboard system uses an adapter pattern to support multiple dashboard implementations:

```
dashboard/
├── interface.go          # Common Dashboard interface
├── factory.go            # Dashboard factory with env configuration
├── terminal_adapter.go   # Terminal progress indicator adapter
├── web_adapter.go        # Web dashboard adapter
└── README.md            # This file

terminal/
└── progress.go          # Terminal progress spinner implementation

web/
├── server.go            # Gin web server
├── models.go            # Dashboard data models
├── handlers.go          # HTTP request handlers
├── websocket.go         # WebSocket support for real-time updates
└── templates/           # HTML templates
    ├── index.html       # Main dashboard page
    └── partials/        # HTMX partial templates
        ├── status.html
        ├── eras.html
        ├── activities.html
        └── errors.html
```

## Web Dashboard Features

- **Real-time Updates**: Uses WebSocket for live data streaming
- **Era Progress**: Visual progress bars for each Cardano era
- **Performance Metrics**: Blocks/sec, memory usage, CPU usage
- **Activity Feed**: Recent blockchain processing activities
- **Error Monitor**: Track and display recent errors
- **Responsive Design**: Works on desktop and mobile
- **HTMX Integration**: Efficient partial page updates

## Terminal Dashboard Features

- **Minimal Output**: Clean spinner with essential metrics
- **SSH Friendly**: Works perfectly over SSH connections
- **Low Overhead**: Minimal resource usage
- **Detailed Mode**: Optional verbose output with `DETAILED_LOG=true`

## Remote Access

When running the web dashboard on a remote VM:

1. Ensure port 8080 (or your chosen port) is open in firewall
2. Access via `http://your-vm-ip:8080`
3. Dashboard updates in real-time without page refresh

## Future Expansion

The web dashboard is designed to be expandable for future features:
- Database explorer functionality
- Transaction search
- Block details viewer
- Pool information
- Stake distribution charts
- Network statistics
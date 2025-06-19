# Dashboard Migration Guide

This guide explains how to migrate from the termdash dashboard to the new modular dashboard system.

## Overview

The new dashboard system provides three options:
1. **Terminal Progress** - Simple, lightweight progress indicator
2. **Web Dashboard** - Modern web-based dashboard with real-time updates
3. **Legacy Termdash** - Original terminal dashboard (still supported)

## Quick Start

### Terminal Progress (Default)
```bash
# Simple progress indicator (default)
./nectar

# Or explicitly set
export NECTAR_DASHBOARD=terminal
./nectar

# Enable detailed view
export NECTAR_TERMINAL_DETAILED=true
./nectar
```

### Web Dashboard
```bash
# Enable web dashboard
export NECTAR_DASHBOARD=web
export NECTAR_WEB_PORT=8080  # Optional, defaults to 8080
./nectar

# Access at http://localhost:8080
```

### Multiple Dashboards
```bash
# Run both terminal and web dashboards
export NECTAR_MULTI_DASHBOARD=true
export NECTAR_WEB_ENABLED=true
./nectar
```

### Legacy Termdash
```bash
# Use original termdash
export NECTAR_DASHBOARD=termdash
./nectar
```

### No Dashboard
```bash
# Disable all dashboards
export NECTAR_DASHBOARD=none
./nectar
```

## Features Comparison

| Feature | Termdash | Terminal | Web |
|---------|----------|----------|-----|
| Era Progress | ✓ (all eras) | ✓ (current) | ✓ (all eras) |
| Performance Metrics | ✓ | ✓ (basic) | ✓ |
| Activity Feed | ✓ | ✓ (status) | ✓ |
| Error Monitor | ✓ | ✓ (count) | ✓ |
| Charts/Graphs | ✓ | ✗ | ✓ |
| Remote Access | ✗ | ✗ | ✓ |
| Mobile Support | ✗ | ✗ | ✓ |
| Resource Usage | High | Low | Medium |

## Terminal Progress

The terminal progress indicator shows:
- Current era being synced
- Total blocks processed
- Sync percentage
- Current blocks/sec rate

Example output:
```
⠋ Syncing Conway Era | Blocks: 1,234,567 | 78.5% | 1234 b/s
```

### Detailed Mode

With `NECTAR_TERMINAL_DETAILED=true`:
```
⠋ Status: Syncing Conway Era
  Progress: [██████████████████░░░░░░░░░░░░] 78.5%
  Blocks: 1,234,567 | Speed: 1234 b/s
```

## Web Dashboard

The web dashboard provides:
- Real-time updates via WebSocket
- Era progress visualization
- Performance metrics and charts
- Activity feed with filtering
- Error monitoring and categorization
- Mobile-responsive design

### API Endpoints

- `GET /` - Main dashboard page
- `GET /api/status` - Current status JSON
- `GET /api/eras` - Era progress data
- `GET /api/performance` - Performance metrics
- `GET /api/activities` - Activity feed
- `GET /api/errors` - Error statistics
- `WS /ws` - WebSocket for real-time updates

### Security Considerations

By default, the web dashboard binds to all interfaces (0.0.0.0). For production:

```bash
# Bind to localhost only
export NECTAR_WEB_BIND=127.0.0.1:8080

# Use reverse proxy (nginx example)
location /nectar/ {
    proxy_pass http://localhost:8080/;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

## Migration Steps

### 1. Update Dependencies
```bash
go get github.com/gin-gonic/gin
go get github.com/gin-contrib/cors
go get github.com/gorilla/websocket
```

### 2. Update Code

Replace dashboard initialization:
```go
// Old
termdashDashboard, err := dashboard.NewDashboard()

// New
dash, err := dashboard.CreateDashboardFromEnv()
indexer.dashboard = dash
```

Update dashboard calls:
```go
// Old
indexer.termdashDashboard.UpdateEraProgress(era, progress)

// New
indexer.dashboard.UpdateEraProgress(era, progress)
```

### 3. Test Migration

1. Run with terminal dashboard first
2. Verify metrics are displayed correctly
3. Enable web dashboard
4. Compare outputs between dashboards
5. Monitor resource usage

## Troubleshooting

### Terminal Issues

**Problem**: No output visible
- Check terminal supports UTF-8
- Try `export TERM=xterm-256color`
- Disable detailed mode

**Problem**: Garbled characters
- Update terminal emulator
- Use ASCII-only mode (future feature)

### Web Dashboard Issues

**Problem**: Cannot connect
- Check firewall rules
- Verify port is not in use
- Check bind address

**Problem**: WebSocket disconnects
- Check proxy configuration
- Increase timeout values
- Monitor server logs

## Performance Considerations

### Resource Usage

- **Termdash**: ~50MB RAM, moderate CPU
- **Terminal**: ~5MB RAM, minimal CPU
- **Web**: ~20MB RAM + browser usage

### Network Overhead

The web dashboard uses WebSocket for updates:
- Initial connection: ~10KB
- Updates: ~1KB/sec average
- Compression supported

## Future Enhancements

Planned features:
1. Database explorer interface
2. SQL query interface
3. Export functionality
4. Custom dashboards
5. Authentication/authorization
6. Metrics API for Prometheus
7. Alert configuration

## Environment Variables Reference

| Variable | Description | Default |
|----------|-------------|---------|
| `NECTAR_DASHBOARD` | Dashboard type (terminal/web/termdash/none) | terminal |
| `NECTAR_TERMINAL_DETAILED` | Enable detailed terminal view | false |
| `NECTAR_WEB_PORT` | Web dashboard port | 8080 |
| `NECTAR_WEB_BIND` | Web bind address | 0.0.0.0 |
| `NECTAR_MULTI_DASHBOARD` | Enable multiple dashboards | false |
| `NECTAR_WEB_ENABLED` | Enable web in multi mode | false |
| `NECTAR_NO_DASHBOARD` | Legacy: disable dashboard | false |

## Support

For issues or questions:
1. Check the logs for errors
2. Verify environment variables
3. Test with minimal configuration
4. Report issues with full context
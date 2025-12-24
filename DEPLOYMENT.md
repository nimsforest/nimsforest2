# Deployment Guide for NimsForest

This guide covers deployment options for NimsForest on Debian-based systems.

## Table of Contents

- [System Requirements](#system-requirements)
- [Debian Package Installation](#debian-package-installation)
- [Docker Deployment](#docker-deployment)
- [Systemd Service](#systemd-service)
- [Configuration](#configuration)
- [NATS Setup](#nats-setup)

## System Requirements

### Minimum Requirements

- **OS**: Debian 11 (Bullseye) or later, Ubuntu 20.04 LTS or later
- **CPU**: 1 core (2+ cores recommended)
- **RAM**: 512 MB (2 GB+ recommended)
- **Disk**: 100 MB for application (+ space for NATS data)
- **Network**: Access to NATS server (local or remote)

### Software Dependencies

- NATS Server 2.10+ with JetStream enabled
- systemd (for service management)

## Debian Package Installation

### Download and Install

Download the latest `.deb` package from the [releases page](https://github.com/yourusername/nimsforest/releases):

```bash
# For AMD64
wget https://github.com/yourusername/nimsforest/releases/download/v1.0.0/nimsforest_1.0.0_amd64.deb

# For ARM64
wget https://github.com/yourusername/nimsforest/releases/download/v1.0.0/nimsforest_1.0.0_arm64.deb

# Install the package
sudo dpkg -i nimsforest_1.0.0_amd64.deb

# Install any missing dependencies
sudo apt-get install -f
```

### Package Contents

The Debian package installs:

- **Binary**: `/usr/local/bin/forest`
- **Config**: `/etc/nimsforest/`
- **Data**: `/var/lib/nimsforest/`
- **Logs**: `/var/log/nimsforest/`
- **Service**: `/usr/lib/systemd/system/nimsforest.service`
- **User**: `forest` system user

### Manage the Service

```bash
# Start the service
sudo systemctl start nimsforest

# Stop the service
sudo systemctl stop nimsforest

# Restart the service
sudo systemctl restart nimsforest

# Enable on boot
sudo systemctl enable nimsforest

# Check status
sudo systemctl status nimsforest

# View logs
sudo journalctl -u nimsforest -f
```

## Docker Deployment

### Using Docker Compose

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  nats:
    image: nats:2.12-alpine
    container_name: nimsforest-nats
    command:
      - "--jetstream"
      - "--store_dir=/data"
      - "--http_port=8222"
    ports:
      - "4222:4222"
      - "8222:8222"
    volumes:
      - nats-data:/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8222/healthz"]
      interval: 10s
      timeout: 5s
      retries: 3

  nimsforest:
    image: yourusername/nimsforest:latest
    container_name: nimsforest
    depends_on:
      nats:
        condition: service_healthy
    environment:
      - NATS_URL=nats://nats:4222
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "pgrep", "-x", "forest"]
      interval: 30s
      timeout: 3s
      retries: 3

volumes:
  nats-data:
```

Deploy with:

```bash
docker-compose up -d
```

### Standalone Docker

```bash
# Pull the image
docker pull yourusername/nimsforest:latest

# Run the container
docker run -d \
  --name nimsforest \
  --restart unless-stopped \
  -e NATS_URL=nats://your-nats-server:4222 \
  yourusername/nimsforest:latest
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/nimsforest.git
cd nimsforest

# Build the Docker image
docker build -t nimsforest:local .

# Run
docker run -d \
  --name nimsforest \
  -e NATS_URL=nats://localhost:4222 \
  nimsforest:local
```

## Systemd Service

If not using the Debian package, you can manually set up the systemd service:

### Create Service File

Create `/etc/systemd/system/nimsforest.service`:

```ini
[Unit]
Description=NimsForest Event Orchestration System
After=network.target nats.service
Wants=nats.service

[Service]
Type=simple
User=forest
Group=forest
WorkingDirectory=/var/lib/nimsforest
Environment="NATS_URL=nats://localhost:4222"
ExecStart=/usr/local/bin/forest
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nimsforest

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/nimsforest /var/log/nimsforest

[Install]
WantedBy=multi-user.target
```

### Create User and Directories

```bash
# Create system user
sudo useradd -r -s /bin/false -d /var/lib/nimsforest forest

# Create directories
sudo mkdir -p /var/lib/nimsforest
sudo mkdir -p /var/log/nimsforest

# Set permissions
sudo chown -R forest:forest /var/lib/nimsforest
sudo chown -R forest:forest /var/log/nimsforest

# Copy binary
sudo cp forest /usr/local/bin/
sudo chmod +x /usr/local/bin/forest

# Reload systemd
sudo systemctl daemon-reload

# Enable and start
sudo systemctl enable nimsforest
sudo systemctl start nimsforest
```

## Configuration

### Environment Variables

NimsForest can be configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_URL` | `nats://localhost:4222` | NATS server connection URL |
| `DEMO` | `false` | Run in demo mode with sample data |

### Configuration File (Optional)

Create `/etc/nimsforest/config.env`:

```bash
NATS_URL=nats://localhost:4222
DEMO=false
```

Update the systemd service to use it:

```ini
[Service]
EnvironmentFile=/etc/nimsforest/config.env
```

## NATS Setup

NimsForest requires a NATS server with JetStream enabled.

### Install NATS on Debian

```bash
# Download and install NATS
curl -sf https://binaries.nats.dev/nats-io/nats-server/v2@latest | sh
sudo mv nats-server /usr/local/bin/

# Create NATS service
sudo tee /etc/systemd/system/nats.service > /dev/null << 'EOF'
[Unit]
Description=NATS Server
After=network.target

[Service]
Type=simple
User=nats
Group=nats
WorkingDirectory=/var/lib/nats
ExecStart=/usr/local/bin/nats-server --jetstream --store_dir=/var/lib/nats --http_port=8222
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Create NATS user and directories
sudo useradd -r -s /bin/false -d /var/lib/nats nats
sudo mkdir -p /var/lib/nats
sudo chown -R nats:nats /var/lib/nats

# Enable and start NATS
sudo systemctl daemon-reload
sudo systemctl enable nats
sudo systemctl start nats

# Verify NATS is running
curl http://localhost:8222/varz
```

### NATS Configuration

For production deployments, create `/etc/nats/nats.conf`:

```conf
# NATS Server Configuration
port: 4222
http_port: 8222

jetstream {
  store_dir: /var/lib/nats
  max_memory_store: 1GB
  max_file_store: 10GB
}

# Enable monitoring
server_name: nimsforest-nats

# Logging
log_file: /var/log/nats/nats.log
logtime: true
```

Update the service to use the config:

```ini
ExecStart=/usr/local/bin/nats-server -c /etc/nats/nats.conf
```

## Production Considerations

### Security

1. **Firewall**: Restrict NATS port (4222) to trusted networks
2. **TLS**: Enable TLS for NATS connections in production
3. **Authentication**: Use NATS authentication tokens or JWT
4. **User Permissions**: Run as non-root `forest` user (handled by package)

### Monitoring

Monitor the application with:

```bash
# View real-time logs
sudo journalctl -u nimsforest -f

# Check service status
sudo systemctl status nimsforest

# Check NATS stats
curl http://localhost:8222/varz
curl http://localhost:8222/jsz
```

### Backup

Back up critical data:

```bash
# Backup NATS JetStream data
sudo tar czf nats-backup-$(date +%Y%m%d).tar.gz /var/lib/nats

# Backup NimsForest data
sudo tar czf nimsforest-backup-$(date +%Y%m%d).tar.gz /var/lib/nimsforest
```

### Updates

Update to a new version:

```bash
# Download new package
wget https://github.com/yourusername/nimsforest/releases/download/v1.1.0/nimsforest_1.1.0_amd64.deb

# Stop the service
sudo systemctl stop nimsforest

# Install new version
sudo dpkg -i nimsforest_1.1.0_amd64.deb

# Start the service
sudo systemctl start nimsforest

# Verify
sudo systemctl status nimsforest
```

## Troubleshooting

### Service Won't Start

```bash
# Check logs
sudo journalctl -u nimsforest -n 50

# Check NATS connectivity
curl http://localhost:8222/varz

# Test binary manually
sudo -u forest /usr/local/bin/forest
```

### Connection Issues

```bash
# Test NATS connection
nc -zv localhost 4222

# Check environment
sudo systemctl show nimsforest --property=Environment
```

### Performance Issues

```bash
# Check resource usage
systemctl status nimsforest
top -u forest

# Check NATS metrics
curl http://localhost:8222/jsz
```

## Support

For issues and questions:
- GitHub Issues: https://github.com/yourusername/nimsforest/issues
- Documentation: https://github.com/yourusername/nimsforest

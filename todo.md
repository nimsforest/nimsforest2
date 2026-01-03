# NATS Sidecar Pattern Implementation

## Overview

Implement a sidecar pattern where each machine running nimsforest has a local NATS node that participates in a full mesh cluster. All nodes are identical with JetStream enabled.

**Key principle**: One node type, one configuration, simple operations.

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Machine A     │     │   Machine B     │     │   Machine C     │
│  ┌───────────┐  │     │  ┌───────────┐  │     │  ┌───────────┐  │
│  │nimsforest │  │     │  │nimsforest │  │     │  │nimsforest │  │
│  └─────┬─────┘  │     │  └─────┬─────┘  │     │  └─────┬─────┘  │
│   localhost     │     │   localhost     │     │   localhost     │
│  ┌─────▼─────┐  │     │  ┌─────▼─────┐  │     │  ┌─────▼─────┐  │
│  │   NATS    │◄─┼─────┼─►│   NATS    │◄─┼─────┼─►│   NATS    │  │
│  │ JetStream │  │     │  │ JetStream │  │     │  │ JetStream │  │
│  └───────────┘  │     │  └───────────┘  │     │  └───────────┘  │
└─────────────────┘     └─────────────────┘     └─────────────────┘

         All nodes identical - full mesh cluster with JetStream
              Streams replicated across nodes (R3 = 3 replicas)
```

### Why Full Mesh?

| Concern | Full Mesh Solution |
|---------|-------------------|
| Complexity | One node type, one config template |
| Failover | Any node can serve any request |
| Data safety | JetStream replicates across nodes |
| Local latency | App connects to localhost |
| Resource cost | ~50-100MB RAM per node (acceptable) |

### Resource Requirements Per Node

| Resource | Usage | Notes |
|----------|-------|-------|
| RAM | 50-100 MB | JetStream enabled |
| CPU | < 1-5% | Mostly idle |
| Disk | Configurable | Based on stream retention |
| Network | Cluster gossip + replication | Lightweight |

---

## Tasks

### Phase 1: Application Connection Resilience

#### 1.1 Add NATS connection options with retry logic
- **File**: `cmd/forest/main.go`
- **Changes**:
  - [ ] Add connection options: `MaxReconnects`, `ReconnectWait`, `DisconnectErrHandler`, `ReconnectHandler`
  - [ ] Implement retry loop on initial connection failure (instead of immediate fatal exit)
  - [ ] Add configurable retry attempts via `NATS_RETRY_ATTEMPTS` env var (default: 10)
  - [ ] Add configurable retry interval via `NATS_RETRY_INTERVAL` env var (default: 5s)
  - [ ] Log connection state changes (disconnected, reconnecting, reconnected)

#### 1.2 Add connection health monitoring
- **File**: `internal/core/connection.go` (new file)
- **Changes**:
  - [ ] Create `ConnectionManager` struct to wrap NATS connection with health state
  - [ ] Implement `IsConnected()` method
  - [ ] Implement `WaitForConnection(ctx, timeout)` method
  - [ ] Add connection event callbacks for monitoring/alerting

#### 1.3 Handle JetStream unavailability gracefully
- **Files**: `internal/core/river.go`, `internal/core/humus.go`, `internal/core/soil.go`
- **Changes**:
  - [ ] Add retry logic when creating streams/buckets
  - [ ] Return meaningful errors when JetStream is unavailable
  - [ ] Consider lazy initialization of JetStream resources

---

### Phase 2: NATS Cluster Configuration

#### 2.1 Create unified cluster node configuration template
- **File**: `scripts/nats/nats-cluster.conf.template` (new file)
- **Contents**:
  ```
  # Server identification
  server_name: ${NODE_NAME}
  
  # Client connections (local only for sidecar)
  port: 4222
  
  # Cluster configuration
  cluster {
    name: nimsforest
    port: 6222
    routes: [
      ${CLUSTER_ROUTES}
    ]
  }
  
  # JetStream configuration
  jetstream {
    store_dir: /var/lib/nats/jetstream
    max_mem: 256MB
    max_file: 10GB
  }
  
  # Monitoring
  http_port: 8222
  ```

#### 2.2 Create configuration generator script
- **File**: `scripts/generate-nats-config.sh` (new file)
- **Features**:
  - [ ] Accept node name and peer addresses as parameters
  - [ ] Generate config from template with substitutions
  - [ ] Validate generated configuration
  - [ ] Support optional TLS configuration
  - [ ] Support optional authentication

---

### Phase 3: Systemd Integration

#### 3.1 Create NATS cluster node systemd service
- **File**: `scripts/systemd/nats.service` (update existing or new)
- **Contents**:
  - [ ] Service definition for NATS cluster node
  - [ ] JetStream enabled
  - [ ] Dependency on network
  - [ ] Restart policy (always restart)
  - [ ] Security hardening

#### 3.2 Update nimsforest service for sidecar dependency
- **File**: `scripts/systemd/nimsforest.service`
- **Changes**:
  - [ ] Change `Wants=nats.service` to `Requires=nats.service`
  - [ ] Add `BindsTo=nats.service` for tight coupling
  - [ ] Ensure proper startup ordering with `After=nats.service`

---

### Phase 4: Setup Scripts

#### 4.1 Create cluster node setup script
- **File**: `scripts/setup-nats-node.sh` (new file)
- **Features**:
  - [ ] Install NATS server binary (if not present)
  - [ ] Accept node name as parameter
  - [ ] Accept peer addresses as parameter (comma-separated)
  - [ ] Generate configuration from template
  - [ ] Install systemd service
  - [ ] Start and enable service
  - [ ] Verify cluster membership

#### 4.2 Update main setup script
- **File**: `scripts/setup-server.sh`
- **Changes**:
  - [ ] Add `--cluster-peers` parameter for peer addresses
  - [ ] Add `--node-name` parameter (default: hostname)
  - [ ] Update NATS setup to use cluster configuration
  - [ ] Add cluster verification step

#### 4.3 Create cluster status script
- **File**: `scripts/cluster-status.sh` (new file)
- **Features**:
  - [ ] Show all cluster members
  - [ ] Show JetStream stream status and replication
  - [ ] Show consumer status
  - [ ] Health check for all nodes

---

### Phase 5: Health & Observability

#### 5.1 Add health check endpoint
- **File**: `cmd/forest/main.go` or `internal/httputil/health.go` (new)
- **Features**:
  - [ ] HTTP endpoint `/health` or `/healthz`
  - [ ] Check NATS connection status
  - [ ] Check JetStream availability
  - [ ] Check cluster membership (connected peers)
  - [ ] Return appropriate HTTP status codes
  - [ ] Include details in response body

#### 5.2 Add metrics for NATS connection (optional)
- **File**: `internal/core/metrics.go` (new file)
- **Metrics**:
  - [ ] `nats_connection_status` (gauge: 0=disconnected, 1=connected)
  - [ ] `nats_reconnect_total` (counter)
  - [ ] `nats_cluster_peers` (gauge)

---

### Phase 6: Documentation

#### 6.1 Create cluster deployment guide
- **File**: `docs/CLUSTER_DEPLOYMENT.md` (new file)
- **Contents**:
  - [ ] Architecture overview with diagrams
  - [ ] Hardware/network requirements
  - [ ] Step-by-step setup for 3-node cluster
  - [ ] Adding/removing nodes
  - [ ] Verification steps
  - [ ] Troubleshooting guide

#### 6.2 Update existing documentation
- **Files**: `README.md`, `DEPLOYMENT.md`, `QUICK_START_GUIDE.md`
- **Changes**:
  - [ ] Document sidecar pattern as default deployment
  - [ ] Update architecture diagrams
  - [ ] Add cluster setup instructions
  - [ ] Document environment variables

---

### Phase 7: Testing

#### 7.1 Add connection resilience tests
- **File**: `internal/core/connection_test.go` (new file)
- **Tests**:
  - [ ] Test reconnection after disconnect
  - [ ] Test initial connection retry
  - [ ] Test connection event callbacks

#### 7.2 Add integration tests for cluster setup
- **File**: `test/e2e/cluster_test.go` (new file)
- **Tests**:
  - [ ] Test cluster formation
  - [ ] Test JetStream replication
  - [ ] Test behavior during node failure
  - [ ] Test message delivery across cluster

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_URL` | `nats://localhost:4222` | Local NATS sidecar URL |
| `NATS_RETRY_ATTEMPTS` | `10` | Initial connection retry attempts |
| `NATS_RETRY_INTERVAL` | `5s` | Interval between retry attempts |
| `NATS_RECONNECT_WAIT` | `2s` | Wait time before reconnect |
| `NATS_MAX_RECONNECTS` | `-1` (infinite) | Max reconnection attempts |

---

## Cluster Configuration Variables

These are used by `setup-nats-node.sh`:

| Variable | Example | Description |
|----------|---------|-------------|
| `NODE_NAME` | `nats-node-1` | Unique name for this node |
| `CLUSTER_PEERS` | `host2:6222,host3:6222` | Comma-separated peer addresses |
| `JETSTREAM_MAX_MEM` | `256MB` | Max memory for JetStream |
| `JETSTREAM_MAX_FILE` | `10GB` | Max disk for JetStream |

---

## Implementation Priority

1. **Phase 1** (Critical): Connection resilience - handle NATS restarts gracefully
2. **Phase 2** (High): Cluster config - enables the sidecar pattern
3. **Phase 3** (High): Systemd integration - production deployment
4. **Phase 4** (High): Setup scripts - easy deployment
5. **Phase 5** (Medium): Health checks - operational visibility
6. **Phase 6** (Medium): Documentation - user guidance
7. **Phase 7** (Low): Testing - validation

---

## Estimated Effort

| Phase | Effort | Dependencies |
|-------|--------|--------------|
| Phase 1 | 2-3 hours | None |
| Phase 2 | 1-2 hours | None |
| Phase 3 | 1 hour | Phase 2 |
| Phase 4 | 2-3 hours | Phase 2, 3 |
| Phase 5 | 2 hours | Phase 1 |
| Phase 6 | 2-3 hours | All above |
| Phase 7 | 2-3 hours | Phase 1, 2 |

**Total estimated effort**: 12-17 hours

---

## Deployment Example

### Setting up a 3-node cluster

```bash
# On machine 1 (192.168.1.10)
sudo ./scripts/setup-nats-node.sh \
  --node-name nats-1 \
  --cluster-peers "192.168.1.11:6222,192.168.1.12:6222"

# On machine 2 (192.168.1.11)
sudo ./scripts/setup-nats-node.sh \
  --node-name nats-2 \
  --cluster-peers "192.168.1.10:6222,192.168.1.12:6222"

# On machine 3 (192.168.1.12)
sudo ./scripts/setup-nats-node.sh \
  --node-name nats-3 \
  --cluster-peers "192.168.1.10:6222,192.168.1.11:6222"
```

### Verifying cluster

```bash
# Check cluster status
nats server list

# Check JetStream status
nats stream list
nats stream info RIVER
```

---

## Notes

- All nodes are identical - same binary, same config structure
- JetStream streams use R3 replication (data on 3 nodes) by default
- For development, single-node mode still works (no peers configured)
- App always connects to `localhost:4222` - simple and predictable
- Consider TLS and authentication for production deployments

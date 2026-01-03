# NATS Sidecar Pattern Implementation

## Overview

Implement a sidecar pattern where each machine running nimsforest has a local NATS leaf node that connects to a central NATS cluster. This provides:

- **Low latency**: Local NATS connection for the application
- **Resilience**: Buffering during network partitions
- **Simplicity**: Application always connects to `localhost:4222`
- **Scalability**: Central cluster handles JetStream persistence

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Machine (per instance)                    │
│  ┌─────────────┐         ┌─────────────────────────────────┐    │
│  │ nimsforest  │ ──────► │ NATS Leaf Node (sidecar)        │    │
│  │ application │  :4222  │ - Lightweight (~15MB RAM)       │    │
│  └─────────────┘         │ - Routes to central cluster     │    │
│                          │ - Local pub/sub works offline   │    │
│                          └───────────────┬─────────────────┘    │
└──────────────────────────────────────────┼──────────────────────┘
                                           │
                              Remote connection to cluster
                                           │
                                           ▼
         ┌─────────────────────────────────────────────────────┐
         │              Central NATS Cluster                    │
         │  ┌─────────┐   ┌─────────┐   ┌─────────┐            │
         │  │ Node 1  │◄─►│ Node 2  │◄─►│ Node 3  │            │
         │  │   JS    │   │   JS    │   │   JS    │            │
         │  └─────────┘   └─────────┘   └─────────┘            │
         │         JetStream with R3 replication               │
         └─────────────────────────────────────────────────────┘
```

---

## Tasks

### Phase 1: Application Connection Resilience

#### 1.1 Add NATS connection options with retry logic
- **File**: `cmd/forest/main.go`
- **Changes**:
  - [ ] Add connection options: `MaxReconnects`, `ReconnectWait`, `DisconnectErrHandler`, `ReconnectHandler`
  - [ ] Implement graceful retry on initial connection failure (instead of immediate fatal exit)
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

### Phase 2: NATS Leaf Node Configuration

#### 2.1 Create leaf node configuration template
- **File**: `scripts/nats/leaf.conf` (new file)
- **Contents**:
  - [ ] Basic leaf node configuration
  - [ ] Remote connection to cluster (configurable URLs)
  - [ ] Local listen address (localhost:4222)
  - [ ] Reconnect settings for cluster connection
  - [ ] JetStream domain configuration (if needed)

#### 2.2 Create cluster node configuration template
- **File**: `scripts/nats/cluster.conf` (new file)
- **Contents**:
  - [ ] JetStream enabled with replication settings
  - [ ] Cluster routes configuration
  - [ ] Leaf node remotes acceptance
  - [ ] Storage directory configuration
  - [ ] Monitoring endpoint

#### 2.3 Create NATS configuration generator script
- **File**: `scripts/generate-nats-config.sh` (new file)
- **Features**:
  - [ ] Generate leaf node config with cluster URLs as parameter
  - [ ] Generate cluster node config with peer addresses
  - [ ] Support for TLS configuration (optional)
  - [ ] Support for authentication (optional)

---

### Phase 3: Systemd Integration

#### 3.1 Create NATS leaf node systemd service
- **File**: `scripts/systemd/nats-leaf.service` (new file)
- **Contents**:
  - [ ] Service definition for local NATS leaf node
  - [ ] Dependency on network
  - [ ] Restart policy (always restart)
  - [ ] Resource limits appropriate for leaf node
  - [ ] Security hardening

#### 3.2 Update nimsforest service for sidecar dependency
- **File**: `scripts/systemd/nimsforest.service`
- **Changes**:
  - [ ] Change `Wants=nats.service` to `Requires=nats-leaf.service`
  - [ ] Add `BindsTo=nats-leaf.service` for tight coupling
  - [ ] Update `After=` to include `nats-leaf.service`

#### 3.3 Create combined sidecar service (optional)
- **File**: `scripts/systemd/nimsforest-with-nats.service` (new file)
- **Features**:
  - [ ] Single service that manages both NATS leaf and nimsforest
  - [ ] Uses `ExecStartPre` to start NATS leaf
  - [ ] Proper shutdown ordering

---

### Phase 4: Setup Scripts

#### 4.1 Create leaf node setup script
- **File**: `scripts/setup-leaf-node.sh` (new file)
- **Features**:
  - [ ] Install NATS server binary (if not present)
  - [ ] Accept cluster URLs as parameter
  - [ ] Generate leaf node configuration
  - [ ] Install systemd service
  - [ ] Start and enable service
  - [ ] Verify connectivity to cluster

#### 4.2 Create cluster setup script
- **File**: `scripts/setup-cluster-node.sh` (new file)
- **Features**:
  - [ ] Install NATS server binary
  - [ ] Accept peer addresses as parameter
  - [ ] Generate cluster configuration with JetStream
  - [ ] Install systemd service
  - [ ] Initialize cluster or join existing

#### 4.3 Update main setup script
- **File**: `scripts/setup-server.sh`
- **Changes**:
  - [ ] Add `--mode` flag: `standalone`, `leaf`, `cluster`
  - [ ] Default to `leaf` mode (sidecar pattern)
  - [ ] Require `--cluster-urls` for leaf mode
  - [ ] Deprecate standalone mode in documentation

---

### Phase 5: Health & Observability

#### 5.1 Add health check endpoint
- **File**: `cmd/forest/main.go` or `internal/httputil/health.go` (new)
- **Features**:
  - [ ] HTTP endpoint `/health` or `/healthz`
  - [ ] Check NATS connection status
  - [ ] Check JetStream availability
  - [ ] Return appropriate HTTP status codes
  - [ ] Include connection details in response

#### 5.2 Add metrics for NATS connection
- **File**: `internal/core/metrics.go` (new file, optional)
- **Metrics**:
  - [ ] `nats_connection_status` (gauge: 0=disconnected, 1=connected)
  - [ ] `nats_reconnect_total` (counter)
  - [ ] `nats_messages_published_total` (counter)
  - [ ] `nats_messages_received_total` (counter)

---

### Phase 6: Documentation

#### 6.1 Create sidecar deployment guide
- **File**: `docs/SIDECAR_DEPLOYMENT.md` (new file)
- **Contents**:
  - [ ] Architecture overview with diagrams
  - [ ] Prerequisites (central cluster setup)
  - [ ] Step-by-step leaf node setup
  - [ ] Verification steps
  - [ ] Troubleshooting guide

#### 6.2 Create cluster setup guide
- **File**: `docs/CLUSTER_SETUP.md` (new file)
- **Contents**:
  - [ ] Hardware requirements for cluster nodes
  - [ ] Network requirements (ports, firewall)
  - [ ] Step-by-step cluster setup (3-node recommended)
  - [ ] JetStream configuration
  - [ ] Monitoring setup

#### 6.3 Update existing documentation
- **Files**: `README.md`, `DEPLOYMENT.md`, `QUICK_START_GUIDE.md`
- **Changes**:
  - [ ] Reference new sidecar pattern as default
  - [ ] Update architecture diagrams
  - [ ] Add links to new documentation

---

### Phase 7: Testing

#### 7.1 Add connection resilience tests
- **File**: `internal/core/connection_test.go` (new file)
- **Tests**:
  - [ ] Test reconnection after disconnect
  - [ ] Test initial connection retry
  - [ ] Test graceful degradation
  - [ ] Test connection event callbacks

#### 7.2 Add integration tests for sidecar setup
- **File**: `test/e2e/sidecar_test.go` (new file)
- **Tests**:
  - [ ] Test leaf node to cluster connectivity
  - [ ] Test message flow through leaf node
  - [ ] Test JetStream operations through leaf
  - [ ] Test behavior during cluster failover

---

## Environment Variables (Final)

| Variable | Default | Description |
|----------|---------|-------------|
| `NATS_URL` | `nats://localhost:4222` | NATS server URL (leaf node) |
| `NATS_RETRY_ATTEMPTS` | `10` | Initial connection retry attempts |
| `NATS_RETRY_INTERVAL` | `5s` | Interval between retry attempts |
| `NATS_RECONNECT_WAIT` | `2s` | Wait time before reconnect attempt |
| `NATS_MAX_RECONNECTS` | `-1` (infinite) | Max reconnection attempts |

---

## Implementation Priority

1. **Phase 1** (Critical): Connection resilience - allows app to handle NATS restarts
2. **Phase 2** (High): NATS configs - enables the sidecar pattern
3. **Phase 3** (High): Systemd integration - production deployment
4. **Phase 4** (Medium): Setup scripts - easier deployment
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

## Notes

- The sidecar pattern assumes a central NATS cluster already exists
- For development, a single NATS instance (standalone mode) is still supported
- Leaf nodes don't store JetStream data - all persistence is on the cluster
- Consider using NATS credentials/TLS for production deployments

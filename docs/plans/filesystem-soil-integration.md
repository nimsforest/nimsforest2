# Filesystem → Soil Integration

**Status**: 📋 Planned
**Goal**: Integrate organizational filesystem as source of truth with Soil KV as performance cache

## 🎯 Objective

Establish the **filesystem as the permanent source of truth** with Soil KV serving as a **fast access cache** for the nimsforest cluster. Enable hundreds of nodes to access shared organizational data (git repos, network drives, object storage) efficiently.

**Architecture Principle:**
```
Filesystem (Source of Truth, External)
    ↓
Soil KV (Cache, Internal, Fast)
```

**Success Criteria:**
- [ ] Filesystem changes detected and propagated to Soil KV cache
- [ ] Agents can access filesystem data (cached or direct)
- [ ] Works across hundreds of distributed nodes
- [ ] Hot data cached in Soil for speed
- [ ] Cold data accessed directly from filesystem
- [ ] Cache invalidation on filesystem changes
- [ ] Optional preloading on startup

---

## 📐 Architecture Overview

### The Correct Model

```
┌─────────────────────────────────────────────────────────────┐
│         Organizational Filesystem (Source of Truth)          │
│  - Git repos (versioned, distributed)                       │
│  - Network drives (NFS/SMB)                                 │
│  - Object storage (S3/GCS/Azure)                            │
│  - Managed by: humans, CI/CD, external systems              │
└────────┬────────────────────────────────────────────────────┘
         │
         │ (filesystem watch / scan)
         ▼
┌─────────────────────────────────────────────────────────────┐
│              Filesystem Source (NEW)                         │
│  - Watches for file changes (fsnotify)                      │
│  - Scans on startup (optional)                              │
│  - Publishes: filesystem.changed events                     │
└────────┬────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│                    River (NATS)                              │
│  Events: file.created, file.modified, file.deleted          │
└────────┬────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│         Tree (Optional Processing)                           │
│  - Parse file content                                       │
│  - Extract metadata                                         │
│  - Transform for caching                                    │
└────────┬────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│              TreeHouse/Nim                                   │
│  - Decide: cache in Soil or access directly?                │
│  - Update Soil KV if caching                                │
│  - Emit cache.updated event                                 │
└────────┬────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│              Soil KV (Cache)                                 │
│  - Hot data: config, metadata, frequently accessed          │
│  - Eventually consistent across cluster                     │
│  - Can be rebuilt from filesystem                           │
└─────────────────────────────────────────────────────────────┘


┌─────────────────────────────────────────────────────────────┐
│         Agents (Nims in Docker)                              │
│  - Mount filesystem directly for cold data                  │
│  - Read from Soil KV for hot data                           │
│  - Write to filesystem (triggers change events)             │
└─────────────────────────────────────────────────────────────┘
```

### Key Insights

1. **Filesystem is eternal** - Survives forest restarts, managed externally
2. **Soil is ephemeral** - Can be rebuilt from filesystem, optimized for speed
3. **Cache what's hot** - Config, metadata, frequently accessed files
4. **Direct access for cold** - Large datasets, media, archives
5. **Changes propagate** - Filesystem changes → events → cache invalidation

---

## 📋 Implementation Tasks

### Task 1: Filesystem Source Type

**Goal**: Create a Source that watches filesystem and emits events

**File**: `/pkg/runtime/source_filesystem.go` (NEW)

**Configuration:**
```yaml
sources:
  # Watch config directory
  config_watcher:
    type: filesystem_watch
    path: /config/**/*.yaml
    publishes: filesystem.config.changed
    load_on_startup: true  # Preload into cache
    debounce: 1s          # Batch rapid changes

  # Watch git repo
  docs_repo:
    type: filesystem_watch
    path: /repos/docs/**/*.md
    publishes: filesystem.docs.changed
    load_on_startup: false  # Too large, access on demand

  # Scan data directory once on startup
  data_loader:
    type: filesystem_scan
    path: /data/**/*.json
    publishes: filesystem.data.scanned
    scan_once: true
```

**Implementation:**
```go
package runtime

import (
    "context"
    "fmt"
    "log"
    "path/filepath"
    "time"

    "github.com/fsnotify/fsnotify"
    "github.com/yourusername/nimsforest/internal/core"
)

// FilesystemSource watches a filesystem path and emits events.
type FilesystemSource struct {
    config   FilesystemSourceConfig
    wind     *core.Wind
    river    *core.River
    watcher  *fsnotify.Watcher
    cancel   context.CancelFunc
}

// FilesystemSourceConfig configures filesystem watching.
type FilesystemSourceConfig struct {
    Name           string
    Type           string // "filesystem_watch" or "filesystem_scan"
    Path           string // Glob pattern (e.g., /config/**/*.yaml)
    Publishes      string
    LoadOnStartup  bool          // Emit events for existing files on startup
    ScanOnce       bool          // Scan once and exit (no watching)
    Debounce       time.Duration // Batch rapid changes
    IgnorePatterns []string      // Globs to ignore (e.g., *.tmp, .git/*)
}

func NewFilesystemSource(cfg FilesystemSourceConfig, wind *core.Wind, river *core.River) (*FilesystemSource, error) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, fmt.Errorf("failed to create filesystem watcher: %w", err)
    }

    return &FilesystemSource{
        config:  cfg,
        wind:    wind,
        river:   river,
        watcher: watcher,
    }, nil
}

func (fs *FilesystemSource) Start(ctx context.Context) error {
    childCtx, cancel := context.WithCancel(ctx)
    fs.cancel = cancel

    // Add paths to watcher
    matches, err := filepath.Glob(fs.config.Path)
    if err != nil {
        cancel()
        return fmt.Errorf("failed to glob path %s: %w", fs.config.Path, err)
    }

    for _, match := range matches {
        if err := fs.watcher.Add(match); err != nil {
            log.Printf("[FilesystemSource:%s] Warning: cannot watch %s: %v", fs.config.Name, match, err)
        }
    }

    // Load existing files on startup if configured
    if fs.config.LoadOnStartup {
        if err := fs.scanAndEmit(); err != nil {
            log.Printf("[FilesystemSource:%s] Warning: scan failed: %v", fs.config.Name, err)
        }
    }

    // If scan_once, we're done
    if fs.config.ScanOnce {
        log.Printf("[FilesystemSource:%s] Scan complete (scan_once mode)", fs.config.Name)
        return nil
    }

    // Watch for changes
    go fs.watchLoop(childCtx)

    log.Printf("[FilesystemSource:%s] Started - watching: %s", fs.config.Name, fs.config.Path)
    return nil
}

func (fs *FilesystemSource) Stop() error {
    if fs.cancel != nil {
        fs.cancel()
    }
    if fs.watcher != nil {
        fs.watcher.Close()
    }
    log.Printf("[FilesystemSource:%s] Stopped", fs.config.Name)
    return nil
}

func (fs *FilesystemSource) watchLoop(ctx context.Context) {
    debounceTimer := time.NewTimer(0)
    <-debounceTimer.C // Drain initial timer

    pendingEvents := make(map[string]fsnotify.Event)

    for {
        select {
        case <-ctx.Done():
            return

        case event, ok := <-fs.watcher.Events:
            if !ok {
                return
            }

            // Ignore patterns
            if fs.shouldIgnore(event.Name) {
                continue
            }

            // Collect events for debouncing
            pendingEvents[event.Name] = event
            debounceTimer.Reset(fs.config.Debounce)

        case err, ok := <-fs.watcher.Errors:
            if !ok {
                return
            }
            log.Printf("[FilesystemSource:%s] Watch error: %v", fs.config.Name, err)

        case <-debounceTimer.C:
            // Emit batched events
            for _, event := range pendingEvents {
                fs.emitEvent(event)
            }
            pendingEvents = make(map[string]fsnotify.Event)
        }
    }
}

func (fs *FilesystemSource) emitEvent(event fsnotify.Event) {
    eventType := ""
    switch {
    case event.Op&fsnotify.Create == fsnotify.Create:
        eventType = "created"
    case event.Op&fsnotify.Write == fsnotify.Write:
        eventType = "modified"
    case event.Op&fsnotify.Remove == fsnotify.Remove:
        eventType = "deleted"
    case event.Op&fsnotify.Rename == fsnotify.Rename:
        eventType = "renamed"
    default:
        return // Ignore other ops
    }

    data := map[string]interface{}{
        "path":  event.Name,
        "event": eventType,
        "timestamp": time.Now().Format(time.RFC3339),
    }

    // Read file content if not deleted
    if eventType != "deleted" {
        content, err := os.ReadFile(event.Name)
        if err == nil {
            data["content"] = string(content)
            data["size"] = len(content)
        }
    }

    jsonData, _ := json.Marshal(data)

    // Publish to River
    if err := fs.river.Publish(fs.config.Publishes, jsonData); err != nil {
        log.Printf("[FilesystemSource:%s] Failed to publish event: %v", fs.config.Name, err)
    }

    log.Printf("[FilesystemSource:%s] Emitted: %s %s", fs.config.Name, eventType, event.Name)
}

func (fs *FilesystemSource) scanAndEmit() error {
    matches, err := filepath.Glob(fs.config.Path)
    if err != nil {
        return err
    }

    for _, path := range matches {
        fs.emitEvent(fsnotify.Event{
            Name: path,
            Op:   fsnotify.Create,
        })
    }

    log.Printf("[FilesystemSource:%s] Scanned %d files", fs.config.Name, len(matches))
    return nil
}

func (fs *FilesystemSource) shouldIgnore(path string) bool {
    for _, pattern := range fs.config.IgnorePatterns {
        matched, _ := filepath.Match(pattern, filepath.Base(path))
        if matched {
            return true
        }
    }
    return false
}
```

**Validation:**
- [ ] Watches filesystem paths correctly
- [ ] Emits events on file create/modify/delete
- [ ] Debouncing works (batches rapid changes)
- [ ] Preloading works (load_on_startup)
- [ ] Scan-once mode works
- [ ] Ignore patterns work
- [ ] Handles errors gracefully

---

### Task 2: Cache Strategy TreeHouse

**Goal**: Decide what to cache in Soil KV vs access directly

**File**: `/scripts/treehouses/cache_strategy.lua` (NEW)

**Purpose:** Process filesystem events and decide caching strategy

**Configuration:**
```yaml
treehouses:
  cache_manager:
    subscribes: filesystem.>  # All filesystem events
    publishes: cache.updated
    script: scripts/treehouses/cache_strategy.lua
```

**Implementation:**
```lua
-- Cache strategy rules
local cache_rules = {
    -- Always cache config files
    { pattern = "%.yaml$", action = "cache", ttl = 3600 },
    { pattern = "%.json$", action = "cache", ttl = 3600 },

    -- Cache small markdown docs
    { pattern = "%.md$", action = "cache_if_small", max_size = 100000 },

    -- Never cache large media
    { pattern = "%.mp4$", action = "direct" },
    { pattern = "%.zip$", action = "direct" },

    -- Default: direct access
    { pattern = ".*", action = "direct" }
}

function process(event)
    local path = event.path
    local content = event.content
    local event_type = event.event

    -- Find matching rule
    for _, rule in ipairs(cache_rules) do
        if string.match(path, rule.pattern) then
            if rule.action == "cache" then
                return cache_file(path, content, rule.ttl)
            elseif rule.action == "cache_if_small" then
                if #content <= rule.max_size then
                    return cache_file(path, content, rule.ttl)
                end
            end
            -- direct: don't cache
            break
        end
    end

    -- No caching, just acknowledge
    return { action = "direct", path = path }
end

function cache_file(path, content, ttl)
    -- Extract cache key from path
    local key = "file:" .. path

    return {
        action = "cache",
        soil_key = key,
        content = content,
        ttl = ttl,
        path = path
    }
end
```

**Nim to Handle Caching:**
```yaml
nims:
  cache_writer:
    subscribes: cache.updated
    publishes: cache.written
    prompt: |
      You receive cache instructions. Write to Soil KV using the Bury operation.
      Return success/failure status.
```

**Validation:**
- [ ] Small files cached
- [ ] Large files skipped
- [ ] Config always cached
- [ ] Cache keys well-formatted
- [ ] TTL respected

---

### Task 3: Agent Filesystem Access

**Goal**: Enable agents to access filesystem (mount network drives)

**File**: `/pkg/runtime/nim.go` (MODIFY)

**Configuration:**
```yaml
nims:
  document_processor:
    subscribes: document.process
    publishes: document.processed
    prompt: scripts/nims/process_doc.md

    # Agent mounts filesystem
    filesystem_access:
      - source: /mnt/docs           # Host path (network drive)
        target: /data/docs          # Container path
        readonly: true
      - source: /mnt/output
        target: /data/output
        readonly: false
```

**Implementation:**
```go
// In Nim Docker container creation:

func (n *Nim) createContainer() error {
    // ... existing code ...

    // Add volume mounts for filesystem access
    binds := []string{}
    for _, fs := range n.config.FilesystemAccess {
        mode := "rw"
        if fs.Readonly {
            mode = "ro"
        }
        bind := fmt.Sprintf("%s:%s:%s", fs.Source, fs.Target, mode)
        binds = append(binds, bind)
    }

    containerConfig := &container.Config{
        // ... existing config ...
    }

    hostConfig := &container.HostConfig{
        Binds: binds,
        // ... existing host config ...
    }

    // ... create container ...
}
```

**Agent Usage (inside Nim container):**
```python
# Inside Nim agent code

# Read from mounted filesystem (direct access)
with open('/data/docs/document.pdf', 'rb') as f:
    content = f.read()

# Or check Soil cache first (via API)
cached = soil_get('file:/mnt/docs/document.pdf')
if cached:
    content = cached
else:
    # Cache miss, read from filesystem
    with open('/data/docs/document.pdf', 'rb') as f:
        content = f.read()
```

**Validation:**
- [ ] Mounts network drives correctly
- [ ] Readonly mounts enforced
- [ ] Agent can read files
- [ ] Agent can write to writable mounts
- [ ] Paths resolve correctly

---

### Task 4: Git Repository Integration

**Goal**: Special handling for git repos (versioned filesystem)

**Configuration:**
```yaml
sources:
  codebase_repo:
    type: git_repository
    url: https://github.com/org/repo.git
    branch: main
    path: /repos/codebase
    publishes: git.codebase.changed

    # Sync strategy
    sync:
      interval: 5m        # git pull every 5 minutes
      on_event: true      # Also pull when webhook received

    # Emit change events
    watch_changes: true   # Watch working tree after pull
```

**Implementation:**
```go
// GitRepositorySource handles git repos specially

type GitRepositorySource struct {
    config   GitSourceConfig
    repoPath string
    wind     *core.Wind
    river    *core.River
}

func (gs *GitRepositorySource) Start(ctx context.Context) error {
    // 1. Clone or open existing repo
    if !gs.repoExists() {
        if err := gs.cloneRepo(); err != nil {
            return err
        }
    }

    // 2. Start periodic pull
    go gs.syncLoop(ctx)

    // 3. Watch for changes in working tree
    if gs.config.WatchChanges {
        go gs.watchFilesystem(ctx)
    }

    return nil
}

func (gs *GitRepositorySource) syncLoop(ctx context.Context) {
    ticker := time.NewTicker(gs.config.SyncInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := gs.gitPull(); err != nil {
                log.Printf("[GitSource:%s] Pull failed: %v", gs.config.Name, err)
            } else {
                gs.emitChanges()
            }
        }
    }
}

func (gs *GitRepositorySource) gitPull() error {
    cmd := exec.Command("git", "-C", gs.repoPath, "pull", "origin", gs.config.Branch)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("git pull failed: %w, output: %s", err, output)
    }
    log.Printf("[GitSource:%s] Pulled: %s", gs.config.Name, strings.TrimSpace(string(output)))
    return nil
}
```

**Validation:**
- [ ] Clones repo on first startup
- [ ] Pulls updates periodically
- [ ] Emits change events after pull
- [ ] Handles merge conflicts gracefully
- [ ] Works with SSH/HTTPS auth

---

### Task 5: Cache Invalidation

**Goal**: Invalidate Soil cache when filesystem changes

**File**: `/scripts/treehouses/cache_invalidator.lua` (NEW)

**Configuration:**
```yaml
treehouses:
  cache_invalidator:
    subscribes: filesystem.>
    publishes: cache.invalidated
    script: scripts/treehouses/cache_invalidator.lua
```

**Implementation:**
```lua
function process(event)
    local path = event.path
    local event_type = event.event
    local cache_key = "file:" .. path

    if event_type == "deleted" then
        -- File deleted, remove from cache
        return {
            action = "delete",
            soil_key = cache_key
        }
    elseif event_type == "modified" then
        -- File modified, invalidate cache
        return {
            action = "invalidate",
            soil_key = cache_key,
            new_content = event.content
        }
    end

    return nil
end
```

**Nim to Handle Invalidation:**
```yaml
nims:
  cache_invalidator:
    subscribes: cache.invalidated
    publishes: cache.clean
    prompt: |
      Delete or update Soil cache entries as instructed.
```

**Validation:**
- [ ] Deleted files removed from cache
- [ ] Modified files re-cached
- [ ] Stale data never served
- [ ] Cache stays consistent

---

## 🔧 Technical Specifications

### Filesystem Types Supported

| Type | Technology | Use Case |
|------|------------|----------|
| **Local Filesystem** | Direct mount | Development, single-node |
| **Network Drive** | NFS/SMB | Shared organizational storage |
| **Git Repository** | git clone/pull | Versioned code/docs |
| **Object Storage** | S3/GCS via FUSE | Cloud-native, scalable |

### Cache Strategy Decision Tree

```
File change detected
  ↓
Is it config? (.yaml, .json)
  YES → Cache in Soil (hot data)
  NO ↓
Is it < 100KB?
  YES → Cache in Soil (small, fast)
  NO ↓
Is it frequently accessed? (check metrics)
  YES → Cache in Soil
  NO ↓
Direct filesystem access (cold data)
```

### Multi-Node Coordination

**Problem:** 100 nodes, same network drive
**Solution:** Each node can:
1. Watch filesystem independently (100 watchers, OK)
2. All emit events to River (NATS handles fanout)
3. All update their local Soil cache
4. Eventual consistency achieved

**OR** (more efficient):
1. ONE node elected as filesystem watcher
2. Publishes events to River
3. All nodes update their Soil cache
4. Less resource usage

---

## ✅ Definition of Done

- [ ] Filesystem Source implemented (watch + scan)
- [ ] Cache strategy TreeHouse implemented
- [ ] Agent filesystem mounting works
- [ ] Git repository integration works
- [ ] Cache invalidation works
- [ ] Configuration schema defined
- [ ] All tests pass
- [ ] Documentation complete
- [ ] Performance acceptable (< 1s cache updates)
- [ ] Works across distributed nodes

---

## 📚 Documentation Requirements

### Architecture Doc (`/docs/architecture/FILESYSTEM_SOIL.md`)

**Contents:**
- Filesystem as source of truth
- Soil KV as cache (ephemeral)
- When to cache vs direct access
- Multi-node coordination
- Cache invalidation strategy
- Performance characteristics

### User Guide Update

Add to `/docs/guides/QUICK_START.md`:
- How to configure filesystem sources
- How to mount network drives in agents
- Cache strategy best practices
- Troubleshooting filesystem access

---

## 🎯 Success Metrics

**Functional:**
- Filesystem changes propagate to Soil cache ✓
- Agents access both cached and direct data ✓
- Works across 100+ nodes ✓
- Cache invalidation < 5s latency ✓

**Performance:**
- Hot data (cached): < 10ms access time ✓
- Cold data (direct): < 100ms access time ✓
- Cache hit rate > 80% for hot data ✓

**Reliability:**
- No cache inconsistency ✓
- Network drive failures handled gracefully ✓
- Git pull failures logged and retried ✓

---

## 🔍 Edge Cases & Error Handling

### 1. Network Drive Unavailable
**Symptom:** Mount fails or disconnects
**Handling:**
- Retry with exponential backoff
- Fall back to cached data if available
- Alert operator
- Continue with degraded service

### 2. Git Merge Conflicts
**Symptom:** `git pull` fails with conflicts
**Handling:**
- Log error
- Don't update working tree
- Alert operator
- Manual resolution required

### 3. Cache Memory Limits
**Symptom:** Soil KV fills up
**Handling:**
- LRU eviction policy
- Monitor cache size
- Configurable max size
- Warn when approaching limit

### 4. Large File Changes
**Symptom:** 10GB file modified
**Handling:**
- Don't cache (direct access only)
- Emit event but skip content
- Stream processing if needed

### 5. Rapid Changes
**Symptom:** File modified 1000x/sec
**Handling:**
- Debouncing (batch events)
- Rate limiting
- Skip intermediate states
- Only cache final state

---

## 🚀 Future Enhancements (Optional)

### Phase 2: Smart Caching
- ML-based cache prediction
- Access pattern analysis
- Automatic cache sizing
- Prefetching frequently accessed files

### Phase 3: Write-Through Cache
- Agents write to Soil
- Automatically sync to filesystem
- Conflict resolution
- Optimistic locking

### Phase 4: Multi-Tier Cache
- L1: In-memory (per-node)
- L2: Soil KV (cluster)
- L3: Filesystem (permanent)

### Phase 5: CDN Integration
- Cache static assets in CDN
- Invalidate on filesystem changes
- Serve public content faster

---

## 📖 References

- `fsnotify` library: https://github.com/fsnotify/fsnotify
- NATS JetStream KV: https://docs.nats.io/nats-concepts/jetstream/key-value-store
- Git integration patterns: https://git-scm.com/book/en/v2
- Cache invalidation strategies: https://martinfowler.com/bliki/TwoHardThings.html

---

**Next Steps**: Begin Task 1 by implementing the Filesystem Source type with fsnotify integration.

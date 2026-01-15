# Bedrock: Persistent Storage Foundation

**Status**: 📋 Planned
**Goal**: Establish Bedrocks as the persistent storage layer beneath Soil

## Core Metaphor

```
┌─────────────────────────────────────────────────────────────┐
│                         Forest                               │
├─────────────────────────────────────────────────────────────┤
│  Wind (ephemeral events)                                    │
│  River (persistent streams)                                 │
│  Trees, TreeHouses, Nims (processing)                       │
│  Songbirds (outputs)                                        │
│  Sources (event inputs)                                     │
├─────────────────────────────────────────────────────────────┤
│                          Soil                                │
│              (KV store - RAM, working memory)                │
│         Fast access, can be rebuilt from Bedrock             │
├─────────────────────────────────────────────────────────────┤
│                        Bedrock                               │
│         (Persistent storage - the foundation)                │
│              Survives crashes, source of truth               │
│                                                             │
│      git          unix       google_drive       s3          │
│    (audited)    (fast)       (external)     (scalable)      │
└─────────────────────────────────────────────────────────────┘
```

**Soil = RAM**: Fast working memory, ephemeral, rebuilt on restart
**Bedrock = Heap**: Persistent storage, survives crashes, source of truth

---

## Bedrock Types

### 1. Git Bedrock (Recommended Default)

Full audit trail, PR workflow for human approval.

```yaml
bedrocks:
  docs:
    type: git
    path: /repos/docs
    remote: git@github.com:org/docs.git
    branch: main
    write_mode: pull_request    # or "commit" for direct
    pr_config:
      base_branch: main
      branch_prefix: nim/
      reviewers: ["@team-leads"]
      labels: ["ai-generated"]
```

**Use for**: Code, documentation, config, anything worth tracking.

**Training data value**: Every PR = proposal/approval pair for fine-tuning.

### 2. Unix Bedrock

Direct filesystem access, no audit trail, fastest writes.

```yaml
bedrocks:
  scratch:
    type: unix
    path: /data/scratch

  cache:
    type: unix
    path: /var/cache/nims
```

**Use for**: Scratch space, caches, high-frequency writes, temp files.

### 3. External Bedrocks

External systems where users work directly. We integrate, don't control.

```yaml
bedrocks:
  sales_docs:
    type: google_drive
    credentials: /secrets/gdrive.json
    root: "Shared Drives/Sales"
    readonly: false
    notify_on_write: slack.sales

  assets:
    type: s3
    bucket: company-assets
    region: us-east-1
    readonly: true
```

**Use for**: Business documents, shared drives, cloud storage.

**Versioning**: Relies on platform's own versioning (Google Drive history, S3 versioning).

---

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────┐
│                    BedrockSource                             │
│            (Storage-specific adapter)                        │
├─────────────────────────────────────────────────────────────┤
│  type: git          → git operations + fsnotify             │
│  type: unix         → fsnotify                              │
│  type: google_drive → Drive API + polling                   │
│  type: s3           → S3 events + polling                   │
├─────────────────────────────────────────────────────────────┤
│  All emit standardized events:                              │
│    bedrock.{name}.mounted                                   │
│    bedrock.{name}.file.created                              │
│    bedrock.{name}.file.modified                             │
│    bedrock.{name}.file.deleted                              │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│              BedrockReadTreehouse                            │
│              (Bedrock → Soil indexing)                       │
├─────────────────────────────────────────────────────────────┤
│  subscribes: bedrock.{name}.>                               │
│  publishes:  index.{name}.updated                           │
│                                                             │
│  .mounted   → scan, build tree, populate index              │
│  .file.*    → update index, manage cache                    │
│                                                             │
│  Writes to Soil:                                            │
│    bedrock:{name}:tree                                      │
│    bedrock:{name}:manifest                                  │
│    bedrock:{name}:file:{path}                               │
│    cache:{name}:{path} (hot content)                        │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│              BedrockWriteTreehouse                           │
│              (Soil → Bedrock persistence)                    │
├─────────────────────────────────────────────────────────────┤
│  subscribes: persist.{name}.>                               │
│  publishes:  persist.{name}.complete|pending|failed         │
│                                                             │
│  .request   → acquire lock, write to bedrock, release       │
│  .delete    → acquire lock, delete from bedrock, release    │
│  .move      → acquire lock, move in bedrock, release        │
│                                                             │
│  For git with PR mode:                                      │
│    → create branch, commit, open PR                         │
│    → hold lock until PR merged/closed                       │
└─────────────────────────────────────────────────────────────┘
```

### Implicit TreeHouse Creation

When a Bedrock is configured, TreeHouses are automatically created:

```yaml
bedrocks:
  docs:
    type: git
    path: /repos/docs

# Automatically creates:
#   - BedrockSource "docs" (emits bedrock.docs.*)
#   - BedrockReadTreehouse (subscribes bedrock.docs.>)
#   - BedrockWriteTreehouse (subscribes persist.docs.>)
```

---

## Soil Structure

### Tree (LLM-readable)

```
bedrock:docs:tree →

docs/ (mounted: /repos/docs, type: git)
├── README.md (2.3KB, modified: 2024-01-15)
├── guides/
│   ├── getting-started.md (5.1KB)
│   └── advanced.md (12KB)
└── api/
    ├── reference.md (45KB)
    └── examples/
        └── auth.md (8KB)

5 files, 3 directories, 72.3KB total
```

Single Soil read → full context for LLMs.

### Manifest

```json
bedrock:docs:manifest → {
  "name": "docs",
  "type": "git",
  "root": "/repos/docs",
  "remote": "git@github.com:org/docs.git",
  "branch": "main",
  "file_count": 5,
  "total_size": 74035,
  "last_scan": "2024-01-15T10:30:00Z"
}
```

### File Metadata

```json
bedrock:docs:file:guides/getting-started.md → {
  "path": "guides/getting-started.md",
  "size": 5120,
  "modified": "2024-01-15T09:00:00Z",
  "type": "text/markdown",
  "content_hash": "sha256:abc123..."
}
```

### Locks

```json
bedrock:docs:lock:guides/getting-started.md → {
  "holder": "nim-writer-xyz",
  "type": "write",           // or "pending_pr"
  "acquired": "2024-01-15T10:30:00Z",
  "ttl": 30,                 // seconds, null for pending_pr
  "pr": null                 // or "org/repo#123" if pending_pr
}
```

### Cache (Hot Content)

```
cache:docs:guides/getting-started.md → "# Getting Started\n\n..."
```

---

## Locking System

Distributed locking via Soil for multi-node coordination.

### Lock Types

| Type | TTL | Use Case |
|------|-----|----------|
| `write` | 30s | Direct write in progress |
| `pending_pr` | None | Awaiting human approval (git PR mode) |

### Write Flow (Direct Mode)

```
1. persist.docs.request { path: "file.md", content: "..." }

2. Acquire lock
   → Bury bedrock:docs:lock:file.md
   → If revision conflict: locked by another, wait or fail

3. Write to bedrock
   → BedrockSource.Write(path, content)

4. Release lock
   → Delete bedrock:docs:lock:file.md

5. Publish result
   → persist.docs.complete { path, success: true }

6. BedrockSource detects change
   → Emits bedrock.docs.file.created
   → ReadTreehouse updates index automatically
```

### Write Flow (PR Mode - Git Only)

```
1. persist.docs.request { path: "file.md", content: "..." }

2. Acquire lock (type: pending_pr)
   → Bury bedrock:docs:lock:file.md { type: "pending_pr" }

3. Create PR
   → git checkout -b nim/update-file-md-abc123
   → Write file, commit
   → gh pr create
   → Update lock with PR reference

4. Publish pending
   → persist.docs.pending { path, pr_url, awaiting: "human_approval" }

5. Wait for PR resolution (webhook or poll)

   5a. PR merged:
       → Release lock
       → persist.docs.complete { path, pr: "merged" }

   5b. PR closed:
       → Release lock
       → persist.docs.rejected { path, pr: "closed" }
```

### Lock Acquisition (Optimistic)

```go
func acquireLock(soil *Soil, key string, lock Lock) error {
    existing, rev, err := soil.Dig(key)

    if err == nil {
        // Lock exists
        if existing.Type == "pending_pr" {
            return ErrAwaitingApproval{PR: existing.PR}
        }
        if time.Since(existing.Acquired) < existing.TTL {
            return ErrLocked{Holder: existing.Holder}
        }
        // Expired, take over with revision check
        return soil.Bury(key, lock, rev)
    }

    // No lock, create with revision 0
    return soil.Bury(key, lock, 0)
}
```

---

## Tree Regeneration Strategy

For active bedrocks, batch tree regeneration:

```lua
local pending_changes = 0
local last_tree_update = 0
local BATCH_COUNT = 5
local MAX_DELAY = 10  -- seconds

function on_file_change(event)
    -- Always update individual file key immediately
    update_file_key(event.path, event.metadata)

    pending_changes = pending_changes + 1

    -- Regenerate tree if threshold reached
    if pending_changes >= BATCH_COUNT or (now() - last_tree_update) > MAX_DELAY then
        regenerate_tree()
        pending_changes = 0
        last_tree_update = now()
    end
end
```

### Tree Size Considerations

| Bedrock Size | Tree Document | Regeneration | Notes |
|--------------|---------------|--------------|-------|
| 100 files | ~5KB | <10ms | Trivial |
| 1,000 files | ~50KB | ~50ms | Fine |
| 10,000 files | ~500KB | ~200ms | Acceptable |
| 100,000+ files | ~5MB+ | seconds | Consider depth limiting |

For very large bedrocks, support depth-limited trees:

```
bedrock:assets:tree:depth=2  → Top 2 levels only
bedrock:assets:tree:full     → Complete (generated on-demand)
```

---

## Recovery

When a cluster restarts, Soil is empty. Bedrocks rebuild it:

```
Cluster crash
     │
     ▼
Cluster restarts
     │
     ▼
BedrockSources mount
     │
     ▼
Each emits bedrock.{name}.mounted
     │
     ▼
ReadTreeHouses scan bedrocks
     │
     ▼
Soil repopulated:
  - Trees rebuilt
  - Manifests regenerated
  - File indexes restored
  - Caches warmed (hot files)
     │
     ▼
Forest operational
```

**Bedrock is always the source of truth. Soil can be rebuilt.**

---

## Implementation Tasks

### Task 1: Core Bedrock Interface

```go
// pkg/runtime/bedrock.go

type Bedrock interface {
    // Lifecycle
    Start(ctx context.Context) error
    Stop() error

    // Read operations
    List(path string) ([]FileInfo, error)
    Read(path string) ([]byte, error)
    Stat(path string) (FileInfo, error)

    // Write operations (optional)
    Write(path string, content []byte) error
    Delete(path string) error
    Move(from, to string) error

    // Capabilities
    IsReadOnly() bool
    SupportsWatch() bool
    Type() string
}
```

### Task 2: Unix Bedrock (First Implementation)

```go
// pkg/runtime/bedrock_unix.go

type UnixBedrock struct {
    config  UnixBedrockConfig
    watcher *fsnotify.Watcher
    river   *core.River
}

// Supports Linux, macOS, BSD via fsnotify
```

### Task 3: Git Bedrock

```go
// pkg/runtime/bedrock_git.go

type GitBedrock struct {
    config    GitBedrockConfig
    repoPath  string
    remote    string
    branch    string
    writeMode string  // "commit" or "pull_request"
}

// Extends UnixBedrock with git operations
```

### Task 4: BedrockReadTreehouse

Built-in treehouse that indexes bedrock to Soil.

### Task 5: BedrockWriteTreehouse

Built-in treehouse that persists from Soil to bedrock with locking.

### Task 6: Google Drive Bedrock (Later)

```go
// pkg/runtime/bedrock_gdrive.go

type GoogleDriveBedrock struct {
    config      GDriveConfig
    client      *drive.Service
    pollInterval time.Duration
}
```

---

## Configuration Reference

```yaml
bedrocks:
  # Git bedrock with PR workflow (recommended for important content)
  docs:
    type: git
    path: /repos/docs
    remote: git@github.com:org/docs.git
    branch: main
    write_mode: pull_request
    pr_config:
      base_branch: main
      branch_prefix: nim/
      reviewers: ["@team"]
      labels: ["ai-generated"]
    cache_policy:
      hot_patterns: ["*.md", "*.yaml"]
      max_file_size: 100KB
    tree_config:
      batch_count: 5
      max_delay: 10s

  # Git bedrock with direct commits (audited but no approval)
  config:
    type: git
    path: /repos/config
    write_mode: commit

  # Unix bedrock (fast, no audit)
  scratch:
    type: unix
    path: /data/scratch

  # Google Drive (external, readonly)
  sales:
    type: google_drive
    credentials: /secrets/gdrive.json
    root: "Shared Drives/Sales"
    readonly: true
    poll_interval: 60s

  # S3 bucket
  assets:
    type: s3
    bucket: company-assets
    region: us-east-1
    readonly: true
```

---

## Success Criteria

- [ ] Unix bedrock implemented with fsnotify
- [ ] Git bedrock implemented with PR workflow
- [ ] BedrockReadTreehouse builds tree/index in Soil
- [ ] BedrockWriteTreehouse handles persist.* with locking
- [ ] Cluster recovery rebuilds Soil from bedrocks
- [ ] Tree regeneration batched for performance
- [ ] Locks prevent concurrent writes
- [ ] PR locks held until human approval

---

## Future Enhancements

### External Bedrocks
- Google Drive
- SharePoint/OneDrive
- S3/GCS/Azure Blob
- Dropbox

### Advanced Features
- Conflict resolution strategies
- Cross-bedrock file moves
- Bedrock mirroring/sync
- Access control per bedrock
- Encryption at rest

### Training Data Pipeline
- Export PR history (proposal → approval pairs)
- Track human modifications to AI proposals
- Build fine-tuning datasets from git history

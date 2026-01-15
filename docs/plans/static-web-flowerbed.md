# Static Web Flowerbed - Public Content Server

**Status**: 📋 Planned
**Goal**: Serve static websites from organizational filesystem to external visitors

## 🎯 Objective

Create a **Flowerbed** component that serves static web content (HTML/CSS/JS) from the organizational filesystem to external HTTP clients. This is where the forest's work "blooms" for the world to see.

**Architectural Role:**
```
Bedrock (source of truth) → Soil (index) → Flowerbed (serves) → External Visitors
         ↑ persistent storage         ↑ events via Wind
```

**Success Criteria:**
- [ ] Serves static content from filesystem via HTTP
- [ ] Auto-reloads when filesystem changes detected
- [ ] Supports multiple sites on one forest
- [ ] Works across distributed nodes (load balancing)
- [ ] Fast serving (< 50ms for cached content)
- [ ] Configurable domains and routing

---

## 📐 Architecture Overview

### The Flowerbed Concept

**Flowerbed** = Output component that exposes forest data to external world

Like a flowerbed in a garden:
- 🌸 **Public-facing** - Visible plot where flowers bloom for visitors
- 🌸 **Fed by Bedrock** - Gets content from Bedrock (persistent foundation)
- 🌸 **Attracts visitors** - Beautiful display that draws people in
- 🌸 **Multiple flowers** - Can host many different blooms (sites)

### Data Flow

```
┌─────────────────────────────────────────────────────────────┐
│                     Bedrock Layer                            │
│  (Source of Truth - persistent storage)                      │
│                                                             │
│  bedrocks:                                                  │
│    marketing_site:  type: unix, path: /sites/marketing      │
│    docs_site:       type: git,  path: /repos/docs           │
│    blog:            type: unix, path: /sites/blog           │
└────────┬─────────────────────────────────────────────────────┘
         │
         │ (1) Bedrock emits file change events
         ▼
┌─────────────────────────────────────────────────────────────┐
│                    Wind (NATS)                               │
│  Events: bedrock.marketing_site.file.modified               │
│          bedrock.docs_site.file.created                     │
└────────┬─────────────────────────────────────────────────────┘
         │
         │ (2) Flowerbed subscribes to bedrock events
         ▼
┌─────────────────────────────────────────────────────────────┐
│              Static Web Flowerbed                            │
│  - HTTP server (port 80/443)                                │
│  - Reads content from Bedrock (via Soil cache)              │
│  - Hot-reloads on bedrock change events                     │
│  - Optional: in-memory cache for serving                    │
└────────┬─────────────────────────────────────────────────────┘
         │
         │ (3) Serves content
         ▼
┌─────────────────────────────────────────────────────────────┐
│            External Visitors                                 │
│  Browsers requesting https://example.com                     │
└─────────────────────────────────────────────────────────────┘
```

**Key Insight:** Flowerbed reads from Bedrock (the foundation), not directly from filesystem.
This maintains the layered architecture where Bedrock is the single source of truth.

### Why NOT a Source/Tree/TreeHouse/Nim?

| Component | Role | Why Flowerbed is Different |
|-----------|------|---------------------------|
| **Source** | Input (external → forest) | Flowerbed is OUTPUT (forest → external) |
| **Tree** | Parser (unstructured → structured) | Flowerbed doesn't parse, it serves |
| **TreeHouse** | Processor (event → event) | Flowerbed doesn't process, it presents |
| **Nim** | AI agent (decision making) | Flowerbed is passive serving |
| **Songbird** | Output to platforms (Telegram/Slack) | Similar! But Flowerbed serves HTTP |

**Flowerbed is like Songbird, but for HTTP instead of messaging platforms.**

---

## 📋 Implementation Tasks

### Task 1: Flowerbed Configuration Schema

**Goal**: Define how to configure static website serving

**File**: `/pkg/runtime/config.go` (MODIFY)

**Configuration Schema:**
```yaml
# forest.yaml

# Bedrock: The foundation layer (source of truth)
# See: filesystem-soil-integration.md for full Bedrock details
bedrocks:
  marketing_site:
    type: unix
    path: /sites/marketing

  docs_site:
    type: git
    path: /repos/docs
    remote: git@github.com:org/docs.git
    branch: main
    write_mode: pull_request

  blog:
    type: unix
    path: /sites/blog

# Output: Serve websites to public (reads from Bedrock)
flowerbeds:
  marketing:
    type: static_web

    # Bedrock source (NOT raw filesystem)
    bedrock: marketing_site
    root: public  # Relative to bedrock path

    # HTTP serving
    domain: example.com
    port: 443
    tls:
      enabled: true
      cert: /certs/example.com.crt
      key: /certs/example.com.key

    # Reload on bedrock changes
    subscribes: bedrock.marketing_site.file.>

    # Caching
    cache:
      enabled: true
      max_size_mb: 100
      ttl: 3600

    # Routing
    index: index.html
    error_pages:
      404: 404.html
      500: 500.html

  docs:
    type: static_web
    bedrock: docs_site
    root: /  # Serve entire bedrock
    domain: docs.example.com
    port: 443
    subscribes: bedrock.docs_site.file.>

  # Multiple sites on same port (virtual hosting)
  blog:
    type: static_web
    bedrock: blog
    root: public
    domain: blog.example.com
    port: 443  # Same port, different domain
    subscribes: bedrock.blog.file.>
```

**Config Struct:**
```go
type Config struct {
    Bedrocks    map[string]BedrockConfig     `yaml:"bedrocks"`    // Foundation layer
    Sources     map[string]SourceConfig      `yaml:"sources"`
    Trees       map[string]TreeConfig        `yaml:"trees"`
    TreeHouses  map[string]TreeHouseConfig   `yaml:"treehouses"`
    Nims        map[string]NimConfig         `yaml:"nims"`
    Songbirds   map[string]SongbirdConfig    `yaml:"songbirds"`
    Flowerbeds  map[string]FlowerbedConfig   `yaml:"flowerbeds"`
    Viewer      *ViewerConfig                `yaml:"viewer,omitempty"`
}

type FlowerbedConfig struct {
    Name string `yaml:"-"` // Set from map key
    Type string `yaml:"type"` // "static_web", "api", etc.

    // Bedrock reference (primary source)
    Bedrock    string            `yaml:"bedrock"`     // Name of bedrock to read from
    Root       string            `yaml:"root"`        // Path relative to bedrock root

    // HTTP serving
    Domain     string            `yaml:"domain"`      // Virtual host domain
    Port       int               `yaml:"port"`        // HTTP port
    TLS        *TLSConfig        `yaml:"tls,omitempty"`
    Subscribes string            `yaml:"subscribes"`  // Bedrock change events
    Cache      *CacheConfig      `yaml:"cache,omitempty"`
    Index      string            `yaml:"index,omitempty"`      // Default: index.html
    ErrorPages map[int]string    `yaml:"error_pages,omitempty"`
    Headers    map[string]string `yaml:"headers,omitempty"`    // Custom headers
}

type TLSConfig struct {
    Enabled bool   `yaml:"enabled"`
    Cert    string `yaml:"cert"` // Path to certificate
    Key     string `yaml:"key"`  // Path to private key
}

type CacheConfig struct {
    Enabled   bool `yaml:"enabled"`
    MaxSizeMB int  `yaml:"max_size_mb"` // Max memory cache size
    TTL       int  `yaml:"ttl"`         // Seconds
}
```

**Validation:**
- [ ] Config schema defined
- [ ] Multiple flowerbeds supported
- [ ] Virtual hosting works (multiple domains, same port)
- [ ] TLS optional but configurable
- [ ] Cache settings validated

---

### Task 2: Static Web Flowerbed Implementation

**Goal**: HTTP server that serves files from filesystem

**File**: `/pkg/runtime/flowerbed_static_web.go` (NEW)

**Implementation:**
```go
package runtime

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "sync"
    "time"

    "github.com/yourusername/nimsforest/internal/core"
)

// StaticWebFlowerbed serves static files over HTTP.
type StaticWebFlowerbed struct {
    config FlowerbedConfig
    wind   *core.Wind
    server *http.Server

    // In-memory cache
    cache      map[string][]byte
    cacheMutex sync.RWMutex
    cacheSize  int64

    cancel context.CancelFunc
}

func NewStaticWebFlowerbed(cfg FlowerbedConfig, wind *core.Wind) (*StaticWebFlowerbed, error) {
    if cfg.Root == "" {
        return nil, fmt.Errorf("root path is required")
    }
    if cfg.Port == 0 {
        cfg.Port = 8080 // Default
    }
    if cfg.Index == "" {
        cfg.Index = "index.html"
    }

    return &StaticWebFlowerbed{
        config: cfg,
        wind:   wind,
        cache:  make(map[string][]byte),
    }, nil
}

func (swf *StaticWebFlowerbed) Start(ctx context.Context) error {
    childCtx, cancel := context.WithCancel(ctx)
    swf.cancel = cancel

    // Create HTTP handler
    mux := http.NewServeMux()
    mux.HandleFunc("/", swf.handleRequest)

    // Configure server
    addr := fmt.Sprintf(":%d", swf.config.Port)
    swf.server = &http.Server{
        Addr:         addr,
        Handler:      swf.withLogging(mux),
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 30 * time.Second,
    }

    // Subscribe to reload events
    if swf.config.Subscribes != "" {
        go swf.watchForReloads(childCtx)
    }

    // Start server
    go func() {
        log.Printf("[Flowerbed:%s] Serving %s at http://%s%s",
            swf.config.Name, swf.config.Root, swf.config.Domain, addr)

        var err error
        if swf.config.TLS != nil && swf.config.TLS.Enabled {
            err = swf.server.ListenAndServeTLS(swf.config.TLS.Cert, swf.config.TLS.Key)
        } else {
            err = swf.server.ListenAndServe()
        }

        if err != nil && err != http.ErrServerClosed {
            log.Printf("[Flowerbed:%s] Server error: %v", swf.config.Name, err)
        }
    }()

    return nil
}

func (swf *StaticWebFlowerbed) Stop() error {
    if swf.cancel != nil {
        swf.cancel()
    }

    if swf.server != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        return swf.server.Shutdown(ctx)
    }

    log.Printf("[Flowerbed:%s] Stopped", swf.config.Name)
    return nil
}

func (swf *StaticWebFlowerbed) handleRequest(w http.ResponseWriter, r *http.Request) {
    // Only allow GET and HEAD
    if r.Method != http.MethodGet && r.Method != http.MethodHead {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Virtual host check
    if swf.config.Domain != "" && r.Host != swf.config.Domain {
        http.Error(w, "Not found", http.StatusNotFound)
        return
    }

    // Resolve file path
    requestPath := r.URL.Path
    if requestPath == "/" {
        requestPath = "/" + swf.config.Index
    }

    filePath := filepath.Join(swf.config.Root, filepath.Clean(requestPath))

    // Security: ensure path is within root
    if !swf.isPathSafe(filePath) {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }

    // Try cache first
    if swf.config.Cache != nil && swf.config.Cache.Enabled {
        if content, found := swf.getCached(filePath); found {
            swf.serveContent(w, r, filePath, content)
            return
        }
    }

    // Read from filesystem
    content, err := os.ReadFile(filePath)
    if err != nil {
        if os.IsNotExist(err) {
            swf.handleError(w, 404)
        } else {
            swf.handleError(w, 500)
        }
        return
    }

    // Cache if enabled
    if swf.config.Cache != nil && swf.config.Cache.Enabled {
        swf.setCached(filePath, content)
    }

    swf.serveContent(w, r, filePath, content)
}

func (swf *StaticWebFlowerbed) serveContent(w http.ResponseWriter, r *http.Request, filePath string, content []byte) {
    // Set content type
    contentType := swf.detectContentType(filePath)
    w.Header().Set("Content-Type", contentType)

    // Custom headers
    for key, value := range swf.config.Headers {
        w.Header().Set(key, value)
    }

    // Write content
    w.WriteHeader(http.StatusOK)
    w.Write(content)
}

func (swf *StaticWebFlowerbed) handleError(w http.ResponseWriter, code int) {
    // Try custom error page
    if errorPage, ok := swf.config.ErrorPages[code]; ok {
        errorPath := filepath.Join(swf.config.Root, errorPage)
        if content, err := os.ReadFile(errorPath); err == nil {
            w.Header().Set("Content-Type", "text/html")
            w.WriteHeader(code)
            w.Write(content)
            return
        }
    }

    // Default error
    http.Error(w, http.StatusText(code), code)
}

func (swf *StaticWebFlowerbed) isPathSafe(filePath string) bool {
    // Must be within root directory
    absRoot, _ := filepath.Abs(swf.config.Root)
    absPath, _ := filepath.Abs(filePath)
    return filepath.HasPrefix(absPath, absRoot)
}

func (swf *StaticWebFlowerbed) detectContentType(filePath string) string {
    ext := filepath.Ext(filePath)
    contentTypes := map[string]string{
        ".html": "text/html",
        ".css":  "text/css",
        ".js":   "application/javascript",
        ".json": "application/json",
        ".png":  "image/png",
        ".jpg":  "image/jpeg",
        ".jpeg": "image/jpeg",
        ".gif":  "image/gif",
        ".svg":  "image/svg+xml",
        ".ico":  "image/x-icon",
        ".pdf":  "application/pdf",
        ".xml":  "application/xml",
        ".txt":  "text/plain",
    }

    if ct, ok := contentTypes[ext]; ok {
        return ct
    }
    return "application/octet-stream"
}

// Cache methods
func (swf *StaticWebFlowerbed) getCached(filePath string) ([]byte, bool) {
    swf.cacheMutex.RLock()
    defer swf.cacheMutex.RUnlock()
    content, found := swf.cache[filePath]
    return content, found
}

func (swf *StaticWebFlowerbed) setCached(filePath string, content []byte) {
    swf.cacheMutex.Lock()
    defer swf.cacheMutex.Unlock()

    // Check cache size limit
    maxSize := int64(swf.config.Cache.MaxSizeMB) * 1024 * 1024
    if swf.cacheSize+int64(len(content)) > maxSize {
        // Simple LRU: clear cache when full (could be more sophisticated)
        swf.cache = make(map[string][]byte)
        swf.cacheSize = 0
    }

    swf.cache[filePath] = content
    swf.cacheSize += int64(len(content))
}

func (swf *StaticWebFlowerbed) clearCache() {
    swf.cacheMutex.Lock()
    defer swf.cacheMutex.Unlock()
    swf.cache = make(map[string][]byte)
    swf.cacheSize = 0
    log.Printf("[Flowerbed:%s] Cache cleared", swf.config.Name)
}

// Watch for reload events
func (swf *StaticWebFlowerbed) watchForReloads(ctx context.Context) {
    sub, err := swf.wind.Subscribe(swf.config.Subscribes, func(leaf core.Leaf) {
        log.Printf("[Flowerbed:%s] Reload triggered by: %s", swf.config.Name, leaf.Subject)
        swf.clearCache()
    })

    if err != nil {
        log.Printf("[Flowerbed:%s] Failed to subscribe to reload events: %v", swf.config.Name, err)
        return
    }

    <-ctx.Done()
    sub.Unsubscribe()
}

func (swf *StaticWebFlowerbed) withLogging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf("[Flowerbed:%s] %s %s %s", swf.config.Name, r.Method, r.URL.Path, time.Since(start))
    })
}
```

**Validation:**
- [ ] Serves static files correctly
- [ ] Virtual hosting works (domain filtering)
- [ ] Cache works and respects size limits
- [ ] Reloads cache on filesystem change events
- [ ] TLS works if configured
- [ ] Custom error pages work
- [ ] Security: path traversal prevented
- [ ] Performance: < 50ms for cached content

---

### Task 3: Integrate Flowerbed into Forest Runtime

**Goal**: Wire up Flowerbed lifecycle in Forest

**File**: `/pkg/runtime/forest.go` (MODIFY)

**Changes:**
```go
type Forest struct {
    config Config

    // ... existing fields ...

    flowerbeds []*StaticWebFlowerbed  // NEW
}

func (f *Forest) Start(ctx context.Context) error {
    // ... existing startup ...

    // Start Flowerbeds
    for name, cfg := range f.config.Flowerbeds {
        cfg.Name = name

        switch cfg.Type {
        case "static_web":
            flowerbed, err := NewStaticWebFlowerbed(cfg, f.wind)
            if err != nil {
                return fmt.Errorf("failed to create flowerbed %s: %w", name, err)
            }
            if err := flowerbed.Start(ctx); err != nil {
                return fmt.Errorf("failed to start flowerbed %s: %w", name, err)
            }
            f.flowerbeds = append(f.flowerbeds, flowerbed)

        default:
            return fmt.Errorf("unknown flowerbed type: %s", cfg.Type)
        }
    }

    // ... rest of startup ...
}

func (f *Forest) Stop() {
    // ... existing shutdown ...

    // Stop Flowerbeds
    for _, flowerbed := range f.flowerbeds {
        flowerbed.Stop()
    }

    // ... rest of shutdown ...
}
```

**Validation:**
- [ ] Flowerbeds start with forest
- [ ] Stop cleanly on shutdown
- [ ] Multiple flowerbeds can run simultaneously
- [ ] Config validation works

---

### Task 4: Multi-Node Coordination

**Goal**: Handle multiple nodes serving same site (load balancing)

**Strategies:**

#### Strategy A: All Nodes Serve (Recommended)
```
Load Balancer (nginx/HAProxy)
    ↓
┌────┴────┬────────┬────────┐
│         │        │        │
Node 1    Node 2   Node 3   Node N
(Flowerbed)(Flowerbed)(Flowerbed)(Flowerbed)
    │         │        │        │
    └─────────┴────────┴────────┘
    All read from same NFS mount: /sites/marketing/
```

**Pros:**
- Simple: no coordination needed
- High availability (any node can serve)
- Scales horizontally
- Standard load balancer handles traffic distribution

**Cons:**
- All nodes must mount same filesystem
- Cache invalidation happens on all nodes

**Implementation:** No changes needed! Just run multiple forest instances with same config.

#### Strategy B: Leader Serves (Single Active)
```
Only ONE node serves HTTP (leader-elected)
Others are hot standby
Failover on leader death
```

**Pros:**
- Simple cache management
- No duplicate work

**Cons:**
- Single point of failure (until failover)
- Doesn't scale horizontally
- More complex (leader election)

**Recommended: Strategy A** - Let external load balancer handle distribution.

---

## 🔧 Technical Specifications

### Virtual Hosting

Multiple domains on same port:
```yaml
flowerbeds:
  site1:
    domain: example.com
    port: 443
    root: /sites/site1/

  site2:
    domain: blog.example.com
    port: 443  # Same port!
    root: /sites/site2/

  site3:
    domain: docs.example.com
    port: 443  # Same port!
    root: /sites/site3/
```

**Implementation:** Check `r.Host` header, route to correct Flowerbed.

### Caching Strategy

**What to cache:**
- Small files (< 1MB): HTML, CSS, JS
- Frequently accessed: index.html, main.css
- Static assets: images, fonts

**What NOT to cache:**
- Large files (> 1MB): videos, archives
- Infrequently accessed: old blog posts
- Dynamic content (if any)

**Cache invalidation:**
- On filesystem change event
- On TTL expiration
- Manual via API (optional)

### Performance Targets

| Metric | Target |
|--------|--------|
| Cached content | < 10ms |
| Uncached (filesystem) | < 50ms |
| Cache hit rate | > 80% |
| Concurrent requests | 1000+ req/s per node |
| Memory usage | < 100MB per site |

---

## ✅ Definition of Done

- [ ] FlowerbedConfig schema defined
- [ ] StaticWebFlowerbed implemented
- [ ] Serves static files from filesystem
- [ ] Virtual hosting works (multiple domains)
- [ ] TLS support works
- [ ] In-memory caching works
- [ ] Cache invalidation on filesystem changes
- [ ] Integrated into Forest runtime
- [ ] Multiple flowerbeds can run simultaneously
- [ ] Load balancing works (multi-node)
- [ ] Configuration documented
- [ ] Tests pass
- [ ] Performance targets met

---

## 📚 Documentation Requirements

### User Guide

**File**: `/docs/guides/STATIC_WEB_FLOWERBED.md`

**Contents:**
- What is a Flowerbed
- How to configure static websites
- Virtual hosting setup
- TLS certificate configuration
- Cache tuning
- Load balancing strategies
- Troubleshooting

### Example Configurations

**Simple single site:**
```yaml
bedrocks:
  mysite:
    type: unix
    path: /sites/mysite

flowerbeds:
  mysite:
    type: static_web
    bedrock: mysite
    root: public
    port: 8080
    subscribes: bedrock.mysite.file.>
```

**Multiple sites with TLS (using Bedrock):**
```yaml
bedrocks:
  marketing:
    type: unix
    path: /sites/marketing

  docs:
    type: git
    path: /repos/docs
    remote: git@github.com:org/docs.git
    write_mode: pull_request

flowerbeds:
  marketing:
    type: static_web
    bedrock: marketing
    root: /
    domain: example.com
    port: 443
    tls:
      enabled: true
      cert: /certs/example.com.crt
      key: /certs/example.com.key
    subscribes: bedrock.marketing.file.>

  docs:
    type: static_web
    bedrock: docs
    root: /
    domain: docs.example.com
    port: 443
    tls:
      enabled: true
      cert: /certs/docs.example.com.crt
      key: /certs/docs.example.com.key
    subscribes: bedrock.docs.file.>
```

---

## 🎯 Success Metrics

**Functional:**
- Serves static sites correctly ✓
- Hot-reloads on filesystem changes ✓
- Multiple sites work on same forest ✓
- TLS works ✓

**Performance:**
- Cached: < 10ms response time ✓
- Uncached: < 50ms response time ✓
- 1000+ req/s per node ✓

**Operational:**
- Load balancing works ✓
- High availability (multi-node) ✓
- Easy configuration ✓

---

## 🔍 Edge Cases & Error Handling

### 1. Filesystem Unavailable
**Symptom:** Network mount fails
**Handling:**
- Return 503 Service Unavailable
- Log error
- Retry mount periodically
- Serve from cache if available

### 2. File Deleted During Request
**Symptom:** File exists at start of request, deleted during read
**Handling:**
- Return 404 if read fails
- Clear from cache
- Log warning

### 3. Cache Memory Limit Reached
**Symptom:** Cache exceeds max_size_mb
**Handling:**
- Clear entire cache (simple LRU)
- Or: evict oldest entries (sophisticated LRU)
- Log cache evictions

### 4. TLS Certificate Expired
**Symptom:** HTTPS connections fail
**Handling:**
- Log error prominently
- Don't start server (fail fast)
- Alert operator

### 5. Port Already in Use
**Symptom:** Another process using port 443
**Handling:**
- Fail startup with clear error
- Suggest checking for other servers
- Don't retry silently

---

## 🚀 Future Enhancements (Optional)

### Phase 2: Dynamic Routing
- SPA routing (serve index.html for all routes)
- URL rewriting
- Redirects configuration
- Reverse proxy capabilities

### Phase 3: Advanced Caching
- CDN integration (CloudFlare/Fastly)
- Edge caching
- Stale-while-revalidate
- Cache warming on startup

### Phase 4: Security
- Rate limiting
- IP allowlist/blocklist
- DDoS protection
- WAF integration

### Phase 5: Metrics
- Request logging
- Performance metrics (Prometheus)
- Cache hit/miss rates
- Bandwidth usage

### Phase 6: More Flowerbed Types
- API Flowerbed (dynamic JSON/REST)
- GraphQL Flowerbed
- WebSocket Flowerbed (real-time)
- gRPC Flowerbed

---

## 📖 References

- Go net/http package: https://pkg.go.dev/net/http
- Virtual hosting: https://en.wikipedia.org/wiki/Virtual_hosting
- HTTP caching: https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching
- TLS configuration: https://pkg.go.dev/crypto/tls

---

## 🌸 The Flowerbed Metaphor

Just as a flowerbed is the visible, cultivated plot where flowers bloom for visitors to admire:

**Flowerbed components:**
- 🌸 Show the forest's work to external visitors
- 🌸 Are fed by Bedrock (the persistent foundation layer)
- 🌸 Attract traffic (visitors come to see the blooms)
- 🌸 Come in different types (static web, APIs, etc.)
- 🌸 Multiple flowerbeds can exist in the same garden (forest)
- 🌸 Can contain many different flowers (websites, endpoints)

**Architectural Dependency:**
```
Bedrock (source of truth) → Soil (index/cache) → Flowerbed (serves)
```
Flowerbed never bypasses Bedrock. See [filesystem-soil-integration.md](./filesystem-soil-integration.md) for Bedrock details.

**StaticWebFlowerbed** is the first type of Flowerbed - it serves static HTML/CSS/JS to the world.

Other future Flowerbeds might include:
- **APIFlowerbed** - REST/GraphQL endpoints
- **WebSocketFlowerbed** - Real-time connections
- **gRPCFlowerbed** - High-performance APIs
- **StreamFlowerbed** - SSE/streaming data

---

**Next Steps**: Begin Task 1 by defining the FlowerbedConfig schema in the configuration system.
